package commando

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"sync"

	"github.com/scrapli/scrapligo/driver/base"
	"github.com/scrapli/scrapligo/driver/core"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/transport"
	log "github.com/sirupsen/logrus"
	"github.com/srl-labs/srlinux-scrapli"
	"gopkg.in/yaml.v2"
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
	errNoPlatformDefined = fmt.Errorf("platform is not set, use --platform | -k <platform> to set one of the supported platforms: %q",
		supportedPlatforms)
	errNoUsernameDefined = errors.New("username was not provided. Use --username | -u to set it")
	errNoPasswordDefined = errors.New("password was not provided. Use --passoword | -p to set it")
	errNoCommandsDefined = errors.New("commands were not provided. Use --commands | -c to set a `::` delimited list of commands to run")

	errInvalidCredentialsName = errors.New("invalid credentials name provided for host")
	errInvalidTransportsName  = errors.New("invalid transport name provided for host")

	errInvalidTransport = errors.New("invalid transport name provided in inventory. Transport should be one of: [standard, system]")
)

const (
	fileOutput   = "file"
	stdoutOutput = "stdout"
	defaultName  = "default"
)

type inventory struct {
	Credentials map[string]*credentials `yaml:",omitempty"`
	Transports  map[string]*transports  `yaml:",omitempty"`
	Devices     map[string]*device      `yaml:"devices,omitempty"`
}

type device struct {
	Platform     string   `yaml:"platform,omitempty"`
	Address      string   `yaml:"address,omitempty"`
	Credentials  string   `yaml:"credentials,omitempty"`
	Transport    string   `yaml:"transport,omitempty"`
	SendCommands []string `yaml:"send-commands,omitempty"`
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
	rCh := make(chan *base.MultiResponse)

	if app.output == fileOutput {
		log.SetOutput(os.Stderr)
		log.Infof("Started sending commands and capturing outputs...")
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(i.Devices))

	for n, d := range i.Devices {
		go app.runCommands(n, d, rCh)

		resp := <-rCh
		go app.outputResult(wg, rw, n, d, resp)
	}

	wg.Wait()

	if app.output == fileOutput {
		log.Infof("outputs have been saved to '%s' directory", app.outDir)
	}

	return nil
}

func (app *appCfg) validTransport(t string) bool {
	switch t {
	case transport.SystemTransportName:
		return true
	case transport.StandardTransportName:
		return true
	default:
		return false
	}
}

func (app *appCfg) loadCredentials(o []base.Option, c string) ([]base.Option, error) {
	creds, ok := app.credentials[c]
	if !ok {
		return o, errInvalidCredentialsName
	}

	if creds.Username != "" {
		o = append(o, base.WithAuthUsername(creds.Username))
	}

	if creds.Password != "" {
		o = append(o, base.WithAuthPassword(creds.Password))
	}

	if creds.SecondaryPassword != "" {
		o = append(o, base.WithAuthSecondary(creds.SecondaryPassword))
	}

	if creds.PrivateKey != "" {
		o = append(o, base.WithAuthPrivateKey(creds.PrivateKey))
	}

	return o, nil
}

func (app *appCfg) loadTransport(o []base.Option, t string) ([]base.Option, error) {
	// default to strict key false and standard transport, so load those into options first
	o = append(o, base.WithTransportType(transport.StandardTransportName), base.WithAuthStrictKey(false))

	transp, ok := app.transports[t]
	if !ok {
		if t == defaultName {
			// default can not exist in the inventory, we already set the default settings above
			return o, nil
		}

		return o, errInvalidTransportsName
	}

	if transp.Port != 0 {
		o = append(o, base.WithPort(transp.Port))
	}

	if transp.StrictKey {
		o = append(o, base.WithAuthStrictKey(transp.StrictKey))
	}

	if transp.SSHConfigFile != "" {
		o = append(o, base.WithSSHConfigFile(transp.SSHConfigFile))
	}

	if transp.TransportType != "" {
		if !app.validTransport(transp.TransportType) {
			return nil, errInvalidTransport
		}

		o = append(o, base.WithTransportType(transp.TransportType))
	}

	return o, nil
}

// loadOptions loads options from the provided inventory.
func (app *appCfg) loadOptions(d *device) ([]base.Option, error) {
	var o []base.Option

	var err error

	c := defaultName

	if d.Credentials != "" {
		c = d.Credentials
	}

	o, err = app.loadCredentials(o, c)
	if err != nil {
		return o, err
	}

	t := defaultName

	if d.Transport != "" {
		t = d.Transport
	}

	o, err = app.loadTransport(o, t)
	if err != nil {
		return o, err
	}

	return o, err
}

func (app *appCfg) runCommands(
	name string,
	d *device,
	rCh chan<- *base.MultiResponse) {
	var driver *network.Driver

	var err error

	o, err := app.loadOptions(d)
	if err != nil {
		log.Errorf("invalid transport type provided %s; error: %+v\n", err, name)
		return
	}

	switch d.Platform {
	case "nokia_srlinux":
		driver, err = srlinux.NewSRLinuxDriver(
			d.Address,
			o...,
		)
	default:
		driver, err = core.NewCoreDriver(
			d.Address,
			d.Platform,
			o...,
		)
	}

	if err != nil {
		log.Errorf("failed to create driver for device %s; error: %+v\n", err, name)
		rCh <- nil

		return
	}

	err = driver.Open()
	if err != nil {
		log.Errorf("failed to open connection to device %s; error: %+v\n", err, name)
		rCh <- nil

		return
	}

	r, err := driver.SendCommands(d.SendCommands)
	if err != nil {
		log.Errorf("failed to send commands to device %s; error: %+v\n", err, name)
		rCh <- nil

		return
	}

	rCh <- r
}

func (app *appCfg) outputResult(
	wg *sync.WaitGroup,
	rw responseWriter,
	name string,
	d *device,
	r *base.MultiResponse) {
	defer wg.Done()

	if err := rw.WriteResponse(r, name, d); err != nil {
		log.Errorf("error while writing the response: %v", err)
	}
}

// filterDevices will remove the devices which names do not match the passed filter.
func filterDevices(i *inventory, f string) {
	if f == "" {
		return
	}

	fRe := regexp.MustCompile(f)

	for n := range i.Devices {
		if !fRe.Match([]byte(n)) {
			delete(i.Devices, n)
		}
	}
}

func (app *appCfg) loadInventoryFromYAML(i *inventory) error {
	yamlFile, err := ioutil.ReadFile(app.inventory)
	if err != nil {
		return err
	}

	err = yaml.UnmarshalStrict(yamlFile, i)
	if err != nil {
		log.Fatal(err)
	}

	filterDevices(i, app.devFilter)

	if len(i.Devices) == 0 {
		return errNoDevices
	}

	app.credentials = i.Credentials
	app.transports = i.Transports

	return nil
}

func (app *appCfg) loadInventoryFromFlags(i *inventory) error {
	if app.platform == "" {
		return errNoPlatformDefined
	}

	if app.username == "" {
		return errNoUsernameDefined
	}

	if app.password == "" {
		return errNoPasswordDefined
	}

	if app.commands == "" {
		return errNoCommandsDefined
	}

	cmds := strings.Split(app.commands, "::")

	i.Devices = map[string]*device{}

	i.Devices[app.address] = &device{
		Platform:     app.platform,
		Address:      app.address,
		SendCommands: cmds,
	}

	return nil
}
