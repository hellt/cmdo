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
	errInvalidTransport  = errors.New("invalid transport name provided in inventory. Transport should be one of: [standard, system]")
)

const (
	fileOutput   = "file"
	stdoutOutput = "stdout"
)

type inventory struct {
	Devices  map[string]*device `yaml:"devices,omitempty"`
	Defaults *defaults          `yaml:",omitempty"`
}

type device struct {
	Platform     string   `yaml:"platform,omitempty"`
	Address      string   `yaml:"address,omitempty"`
	Username     string   `yaml:"username,omitempty"`
	Password     string   `yaml:"password,omitempty"`
	SendCommands []string `yaml:"send-commands,omitempty"`
	Extras       *extras  `yaml:",omitempty"`
}

type extras struct {
	Port              int    `yaml:"port,omitempty"`
	SecondaryPassword string `yaml:"secondary-password,omitempty"`
	SSHConfigFile     string `yaml:"ssh-config-file,omitempty"`
	StrictKey         bool   `yaml:"strict-key,omitempty"`
	TransportType     string `yaml:"transport,omitempty"`
}

type defaults struct {
	Username string  `yaml:"username,omitempty"`
	Password string  `yaml:"password,omitempty"`
	Extras   *extras `yaml:",omitempty"`
}

type appCfg struct {
	inventory         string    // path to inventory file
	inventoryDefaults *defaults // loaded defaults settings
	output            string    // output mode
	timestamp         bool      // append timestamp to output dir
	outDir            string    // output directory path
	devFilter         string    // pattern
	platform          string    // platform name
	address           string    // device address
	username          string    // ssh username
	password          string    // ssh password
	commands          string    // commands to send
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

// loadOptionsExtras load the extras from the root of inventory or a specific device.
func (app *appCfg) loadOptionsExtras(e *extras, o []base.Option) ([]base.Option, error) {
	if e.Port != 0 {
		o = append(o, base.WithPort(e.Port))
	}

	if e.SecondaryPassword != "" {
		o = append(o, base.WithAuthSecondary(e.SecondaryPassword))
	}

	if e.SSHConfigFile != "" {
		o = append(o, base.WithSSHConfigFile(e.SSHConfigFile))
	}

	if e.StrictKey {
		o = append(o, base.WithAuthStrictKey(e.StrictKey))
	}

	if e.TransportType != "" {
		if !app.validTransport(e.TransportType) {
			return nil, errInvalidTransport
		}

		o = append(o, base.WithPort(e.Port))
	}

	return o, nil
}

func (app *appCfg) loadOptionsDefaults(o []base.Option) ([]base.Option, error) {
	// load defaults first, if there are more specific settings they will come after and
	// override the defaults, this way we don't need to think about merging things.
	var err error

	if app.inventoryDefaults.Username != "" {
		o = append(o, base.WithAuthUsername(app.inventoryDefaults.Username))
	}

	if app.inventoryDefaults.Password != "" {
		o = append(o, base.WithAuthUsername(app.inventoryDefaults.Password))
	}

	if app.inventoryDefaults.Extras != nil {
		o, err = app.loadOptionsExtras(app.inventoryDefaults.Extras, o)
	}

	return o, err
}

// loadOptions loads options from the provided inventory.
func (app *appCfg) loadOptions(d *device) ([]base.Option, error) {
	var o []base.Option

	// defaulting to auth strict key false
	o = append(o, base.WithAuthStrictKey(false))

	var err error

	if app.inventoryDefaults != nil {
		o, err = app.loadOptionsDefaults(o)
		if err != nil {
			return o, err
		}
	}

	if d.Username != "" {
		o = append(o, base.WithAuthUsername(d.Username))
	}

	if d.Password != "" {
		o = append(o, base.WithAuthPassword(d.Password))
	}

	o, err = app.loadOptionsExtras(d.Extras, o)

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
		return
	}

	err = driver.Open()
	if err != nil {
		log.Errorf("failed to open connection to device %s; error: %+v\n", err, name)
		return
	}

	r, err := driver.SendCommands(d.SendCommands)
	if err != nil {
		log.Errorf("failed to send commands to device %s; error: %+v\n", err, name)
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

	if err := rw.WriteResponse(r, name, d, app); err != nil {
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

	app.inventoryDefaults = i.Defaults

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
		Username:     app.username,
		Password:     app.password,
		SendCommands: cmds,
	}

	return nil
}
