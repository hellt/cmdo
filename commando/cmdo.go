package commando

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/scrapli/scrapligocfg/response"

	"github.com/scrapli/scrapligocfg"

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
	Platform             string          `yaml:"platform,omitempty"`
	Address              string          `yaml:"address,omitempty"`
	Credentials          string          `yaml:"credentials,omitempty"`
	Transport            string          `yaml:"transport,omitempty"`
	SendCommands         []string        `yaml:"send-commands,omitempty"`
	SendCommandsFromFile string          `yaml:"send-commands-from-file,omitempty"`
	SendConfigs          []string        `yaml:"send-configs,omitempty"`
	SendConfigsFromFile  string          `yaml:"send-configs-from-file,omitempty"`
	CfgOperations        []*cfgOperation `yaml:"cfg-operations,omitempty"`
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
	TransportType string `yaml:"transport-type,omitempty"`
}

type cfgOperation struct {
	OperationType  string `yaml:"type,omitempty"`
	Source         string `yaml:"source,omitempty"`
	Config         string `yaml:"config,omitempty"`
	ConfigFromFile string `yaml:"config-from-file,omitempty"`
	Replace        bool   `yaml:"replace,omitempty"`
	Diff           bool   `yaml:"diff,omitempty"`
	Commit         bool   `yaml:"commit,omitempty"`
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

type respTuple struct {
	name string
	resp []interface{}
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

	respCh := make(chan respTuple)

	doneCh := make(chan interface{})

	if app.output == fileOutput {
		log.SetOutput(os.Stderr)
		log.Infof("Started sending commands and capturing outputs...")
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(i.Devices))

	for n, d := range i.Devices {
		go app.runOperations(n, d, respCh)
	}

	go app.outputResult(wg, rw, respCh, doneCh)

	wg.Wait()

	doneCh <- nil

	if app.output == fileOutput {
		log.Infof("outputs have been saved to '%s' directory", app.outDir)
	}

	return nil
}

func runCfgGetConfig(
	name string,
	c *scrapligocfg.Cfg,
	op *cfgOperation,
) (*response.Response, error) {
	source := "running"
	if op.Source != "" {
		source = op.Source
	}

	r, err := c.GetConfig(source)
	if err != nil {
		log.Errorf("get-config operation failed for device %s; error: %+v\n", name, err)

		return nil, err
	}

	return r, nil
}

func runCfgLoadConfig(name string, c *scrapligocfg.Cfg, op *cfgOperation) ([]interface{}, error) {
	var responses []interface{}

	var r *response.Response

	var err error

	if op.Config != "" {
		_, err = c.LoadConfig(op.Config, op.Replace)
	} else if op.ConfigFromFile != "" {
		_, err = c.LoadConfigFromFile(op.ConfigFromFile, op.Replace)
	}

	if err != nil {
		log.Errorf("load-config operation failed for device %s; error: %+v\n", name, err)

		return nil, err
	}

	if op.Diff {
		dr, diffErr := c.DiffConfig("running")
		if diffErr != nil {
			log.Errorf("diff-config operation failed for device %s; error: %+v\n", name, diffErr)

			return nil, diffErr
		}

		responses = append(responses, dr)
	}

	if op.Commit {
		r, err = c.CommitConfig()
		if err != nil {
			log.Errorf("commit-config operation failed for device %s; error: %+v\n", name, err)

			return nil, err
		}

		responses = append(responses, r)
	} else {
		_, err = c.AbortConfig()
		if err != nil {
			log.Errorf("abort-config operation failed for device %s; error: %+v\n", name, err)

			return nil, err
		}
	}

	return responses, nil
}

func runCfg(name string, d *device, driver *network.Driver) ([]interface{}, error) {
	if d.CfgOperations == nil {
		return nil, nil
	}

	c, err := scrapligocfg.NewCfg(driver, d.Platform)
	if err != nil {
		log.Errorf("failed to create cfg connection for device %s; error: %+v\n", name, err)

		return nil, err
	}

	err = c.Prepare()
	if err != nil {
		log.Errorf("failed to prepare cfg session for device %s; error: %+v\n", name, err)

		return nil, err
	}

	var responses []interface{}

	for _, op := range d.CfgOperations {
		switch op.OperationType {
		case "get-config":
			r, opErr := runCfgGetConfig(name, c, op)
			if opErr != nil {
				return nil, opErr
			}

			responses = append(responses, r)
		case "load-config":
			r, opErr := runCfgLoadConfig(name, c, op)
			if opErr != nil {
				return nil, opErr
			}

			responses = append(responses, r...)
		default:
			log.Errorf("invalid operation type '%s' for device %s\n", op.OperationType, name)
		}
	}

	return responses, nil
}

func runConfigs(name string, d *device, driver *network.Driver) error {
	// when sending configs we do not print any responses, as typically configs do not produce any output
	if d.SendConfigsFromFile != "" {
		_, err := driver.SendConfigsFromFile(d.SendConfigsFromFile)
		if err != nil {
			log.Errorf("failed to send configs to device %s; error: %+v\n", name, err)

			return err
		}
	}

	if len(d.SendConfigs) != 0 {
		_, err := driver.SendConfigs(d.SendConfigs)
		if err != nil {
			log.Errorf("failed to send configs to device %s; error: %+v\n", name, err)

			return err
		}
	}

	return nil
}

func runCommands(name string, d *device, driver *network.Driver) ([]interface{}, error) {
	var responses []interface{}

	if d.SendCommandsFromFile != "" {
		r, err := driver.SendCommandsFromFile(d.SendCommandsFromFile)
		if err != nil {
			log.Errorf("failed to send commands to device %s; error: %+v\n", name, err)

			return nil, err
		}

		responses = append(responses, r)
	}

	if len(d.SendCommands) != 0 {
		r, err := driver.SendCommands(d.SendCommands)
		if err != nil {
			log.Errorf("failed to send commands to device %s; error: %+v\n", name, err)

			return nil, err
		}

		responses = append(responses, r)
	}

	return responses, nil
}

func (app *appCfg) runOperations(
	name string,
	d *device,
	rCh chan<- respTuple) {
	driver, err := app.openCoreConn(name, d)
	if err != nil {
		rCh <- respTuple{
			name: name,
			resp: nil,
		}

		return
	}

	var responses []interface{}

	cfgResponses, err := runCfg(name, d, driver)
	if err != nil {
		rCh <- respTuple{
			name: name,
			resp: nil,
		}

		return
	}

	responses = append(responses, cfgResponses...)

	err = runConfigs(name, d, driver)
	if err != nil {
		rCh <- respTuple{
			name: name,
			resp: nil,
		}

		return
	}

	cmdResponses, err := runCommands(name, d, driver)
	if err != nil {
		rCh <- respTuple{
			name: name,
			resp: nil,
		}

		return
	}

	responses = append(responses, cmdResponses...)

	rCh <- respTuple{
		name: name,
		resp: responses,
	}
}

func (app *appCfg) outputResult(
	wg *sync.WaitGroup,
	rw responseWriter,
	rCh chan respTuple,
	doneCh chan interface{},
) {
	for {
		select {
		case <-doneCh:
			return
		case r := <-rCh:
			if err := rw.WriteResponse(r.resp, r.name); err != nil {
				log.Errorf("error while writing the response: %v", err)

				// don't defer the wg.Done because it needs to always be decremented at each
				// iteration!
				wg.Done()
			} else {
				wg.Done()
			}
		}
	}
}
