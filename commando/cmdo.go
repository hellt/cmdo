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

var version string
var commit string
var supportedPlatforms = []string{
	"arista_eos",
	`cisco_iosxr`,
	`cisco_iosxe`,
	`cisco_nxos`,
	`juniper_junos`,
	`nokia_sros`,
	`nokia_sros_classic`,
	`nokia_srlinux`,
}

type inventory struct {
	Devices map[string]device `yaml:"devices,omitempty"`
}

type device struct {
	Platform     string   `yaml:"platform,omitempty"`
	Address      string   `yaml:"address,omitempty"`
	Username     string   `yaml:"username,omitempty"`
	Password     string   `yaml:"password,omitempty"`
	SendCommands []string `yaml:"send-commands,omitempty"`
}

type appCfg struct {
	inventory string // path to inventory file
	output    string // output mode
	timestamp bool   // append timestamp to output dir
	outDir    string // output directory path
	devFilter string // pattern
	platform  string // platform name
	address   string // device address
	username  string // ssh username
	password  string // ssh password
	commands  string // commands to send
}

// run runs the commando
func (app *appCfg) run() error {
	// logging.SetDebugLogger(log.Print)
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

	rw, err := app.newResponseWriter(app.output)
	if err != nil {
		return err
	}

	rCh := make(chan *base.MultiResponse)

	if app.output == "file" {
		log.SetOutput(os.Stderr)
		log.Infof("Started sending commands and capturing outputs...")
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(i.Devices))
	for n, d := range i.Devices {
		go app.runCommands(wg, n, d, rCh)

		resp := <-rCh
		go app.outputResult(wg, rw, n, d, resp)
	}

	wg.Wait()

	if app.output == "file" {
		log.Infof("outputs have been saved to '%s' directory", app.outDir)
	}

	return nil
}

func (app *appCfg) runCommands(wg *sync.WaitGroup, name string, d device, rCh chan<- *base.MultiResponse) {
	var driver *network.Driver
	var err error

	switch d.Platform {
	case "nokia_srlinux":
		driver, err = srlinux.NewSRLinuxDriver(
			d.Address,
			base.WithAuthStrictKey(false),
			base.WithAuthUsername(d.Username),
			base.WithAuthPassword(d.Password),
			base.WithTransportType(transport.StandardTransportName),
		)
	default:
		driver, err = core.NewCoreDriver(
			d.Address,
			d.Platform,
			base.WithAuthStrictKey(false),
			base.WithAuthUsername(d.Username),
			base.WithAuthPassword(d.Password),
			base.WithTransportType(transport.StandardTransportName),
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

func (app *appCfg) outputResult(wg *sync.WaitGroup, rw responseWriter, name string, d device, r *base.MultiResponse) {
	defer wg.Done()
	rw.WriteResponse(r, name, d, app)
}

// filterDevices will remove the devices which names do not match the passed filter
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
		return errors.New("no devices to send commands to")
	}

	return nil
}

func (app *appCfg) loadInventoryFromFlags(i *inventory) error {

	if app.platform == "" {
		return fmt.Errorf("platform is not set, use --platform | -k <platform> to set one of the supported platforms: %q", supportedPlatforms)
	}
	if app.username == "" {
		return errors.New("username was not provided. Use --username | -u to set it")
	}
	if app.password == "" {
		return errors.New("password was not provided. Use --passoword | -p to set it")
	}
	if app.commands == "" {
		return errors.New("commands were not provided. Use --commands | -c to set a `::` delimited list of commands to run")
	}

	cmds := strings.Split(app.commands, "::")

	i.Devices = map[string]device{}

	i.Devices[app.address] = device{
		Platform:     app.platform,
		Address:      app.address,
		Username:     app.username,
		Password:     app.password,
		SendCommands: cmds,
	}

	return nil
}
