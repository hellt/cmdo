package commando

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/scrapli/scrapligo/driver/base"
	"github.com/scrapli/scrapligo/driver/network"
	log "github.com/sirupsen/logrus"
)

var (
	version            string
	commit             string      //nolint:gochecknoglobals
	supportedPlatforms = []string{ //nolint:gochecknoglobals
		"arista_eos",
		`cisco_iosxr`,
		`cisco_iosxe`,
		`cisco_nxos`,
		`juniper_junos`,
		`nokia_sros`,
		`nokia_sros_classic`,
		`nokia_srlinux`,
	}

	errNoDevices         = errors.New("no devices to send commands to")
	errNoPlatformDefined = fmt.Errorf(
		"platform is not set, use --platform | -k <platform> to set one of the supported platforms: %q",
		supportedPlatforms,
	)
	errNoUsernameDefined = errors.New("username was not provided. Use --username | -u to set it")
	errNoPasswordDefined = errors.New("password was not provided. Use --passoword | -p to set it")
	errNoCommandsDefined = errors.New(
		"commands were not provided. Use --commands | -c to set a `::` delimited list of commands to run",
	)

	errInvalidCredentialsName = errors.New("invalid credentials name provided for host")
	errInvalidTransportsName  = errors.New("invalid transport name provided for host")

	errInvalidTransport = errors.New(
		"invalid transport name provided in inventory. Transport should be one of: [standard, system]",
	)
)

const (
	fileOutput   = "file"
	stdoutOutput = "stdout"
	defaultName  = "default"
)

type inventory struct {
	Credentials map[string]*credentials `yaml:"credentials,omitempty"`
	Transports  map[string]*transports  `yaml:"transports,omitempty"`
	Devices     map[string]*device      `yaml:"devices,omitempty"`
}

type device struct {
	Platform             string     `yaml:"platform,omitempty"`
	Address              string     `yaml:"address,omitempty"`
	Credentials          string     `yaml:"credentials,omitempty"`
	Transport            string     `yaml:"transport,omitempty"`
	SendCommands         []string   `yaml:"send-commands,omitempty"`
	SendCommandsFromFile string     `yaml:"send-commands-from-file,omitempty"`
	SendConfigs          []string   `yaml:"send-configs,omitempty"`
	SendConfigsFromFile  string     `yaml:"send-configs-from-file,omitempty"`
	CfgConfig            *cfgConfig `yaml:"cfg-configs,omitempty"`
}

type credentials struct {
	Username          string `yaml:"username,omitempty"`
	Password          string `yaml:"password,omitempty"`
	SecondaryPassword string `yaml:"secondary-password,omitempty"`
	PrivateKey        string `yaml:"private-key,omitempty"`
}

type transports struct {
	Port          int    `yaml:"port,omitempty"`
	StrictKey     bool   `yaml:"strict-key,omitempty"`
	SSHConfigFile string `yaml:"ssh-config-file,omitempty"`
	TransportType string `yaml:"transport,omitempty"`
}

type cfgConfig struct {
	Config         string
	ConfigFromFile string
	Replace        bool
	Diff           bool
	Commit         bool
	Abort          bool
}

type appCfg struct {
	inventory   string                  // path to inventory file
	credentials map[string]*credentials // credentials loaded from inventory
	transports  map[string]*transports  // transports loaded from inventory
	output      string                  // output mode
	timestamp   bool                    // append timestamp to output dir
	outDir      string                  // output directory path
	devFilter   string                  // pattern
	platform    string                  // platform name
	address     string                  // device address
	username    string                  // ssh username
	password    string                  // ssh password
	commands    string                  // commands to send
}

// run runs the commando.
func (app *appCfg) run() error {
	i := &inventory{}
	// start bulk commands routine
	if app.address == "" {
		if err := app.loadInventoryFromYAML(i); err != nil {
			return err
		}
	} else { // else we run commands against a single device
		if err := app.loadInventoryFromFlags(i); err != nil {
			return err
		}
	}

	rw := app.newResponseWriter(app.output)
	rCh := make(chan []*base.MultiResponse)

	if app.output == fileOutput {
		log.SetOutput(os.Stderr)
		log.Infof("Started sending commands and capturing outputs...")
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(i.Devices))

	for n, d := range i.Devices {
		go app.runOperations(n, d, rCh)

		resps := <-rCh
		go app.outputResult(wg, rw, n, resps)
	}

	wg.Wait()

	if app.output == fileOutput {
		log.Infof("outputs have been saved to '%s' directory", app.outDir)
	}

	return nil
}

func runConfigs(name string, d *device, driver *network.Driver) error {
	// when sending configs we do not print any responses, as typically configs do not produce any output
	if d.SendConfigsFromFile != "" {
		_, err := driver.SendConfigsFromFile(d.SendConfigsFromFile)
		if err != nil {
			log.Errorf("failed to send configs to device %s; error: %+v\n", err, name)

			return err
		}
	}

	if len(d.SendConfigs) != 0 {
		_, err := driver.SendConfigs(d.SendConfigs)
		if err != nil {
			log.Errorf("failed to send configs to device %s; error: %+v\n", err, name)

			return err
		}
	}

	return nil
}

func runCommands(name string, d *device, driver *network.Driver) ([]*base.MultiResponse, error) {
	var responses []*base.MultiResponse

	if d.SendCommandsFromFile != "" {
		r, err := driver.SendCommandsFromFile(d.SendCommandsFromFile)
		if err != nil {
			log.Errorf("failed to send commands to device %s; error: %+v\n", err, name)

			return nil, err
		}

		responses = append(responses, r)
	}

	if len(d.SendCommands) != 0 {
		r, err := driver.SendCommands(d.SendCommands)
		if err != nil {
			log.Errorf("failed to send commands to device %s; error: %+v\n", err, name)

			return nil, err
		}

		responses = append(responses, r)
	}

	return responses, nil
}

func (app *appCfg) runOperations(
	name string,
	d *device,
	rCh chan<- []*base.MultiResponse) {
	driver, err := app.openCoreConn(name, d)
	if err != nil {
		rCh <- nil

		return
	}

	err = runConfigs(name, d, driver)
	if err != nil {
		rCh <- nil

		return
	}

	responses, err := runCommands(name, d, driver)
	if err != nil {
		rCh <- nil

		return
	}

	rCh <- responses
}

func (app *appCfg) outputResult(
	wg *sync.WaitGroup,
	rw responseWriter,
	name string,
	r []*base.MultiResponse) {
	defer wg.Done()

	if err := rw.WriteResponse(r, name); err != nil {
		log.Errorf("error while writing the response: %v", err)
	}
}
