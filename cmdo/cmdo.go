package cmdo

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"sync"

	"github.com/scrapli/scrapligo/driver/base"
	"github.com/scrapli/scrapligo/driver/core"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/transport"
	log "github.com/sirupsen/logrus"
	"github.com/srl-labs/srlinux-scrapli"
	"gopkg.in/yaml.v2"
)

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
}

func CLI(args []string) int {
	var app appCfg
	err := app.fromArgs(args)
	if err != nil {
		return 2
	}
	if err = app.run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		return 1
	}
	return 0
}

func (app *appCfg) fromArgs(args []string) error {
	fl := flag.NewFlagSet("cmdo", flag.ContinueOnError)
	fl.StringVar(&app.inventory, "i", "inventory.yml", "path to the inventory file")
	fl.StringVar(&app.output, "o", "file", "print output to: [file, stdout]")
	fl.BoolVar(&app.timestamp, "t", false, "append timestamp to output directory")
	if err := fl.Parse(args); err != nil {
		return err
	}

	return nil
}

func (app *appCfg) run() error {
	// logging.SetDebugLogger(log.Print)
	c := &inventory{}
	yamlFile, err := ioutil.ReadFile(app.inventory)
	if err != nil {
		return err
	}

	err = yaml.UnmarshalStrict(yamlFile, c)
	if err != nil {
		log.Fatal(err)
	}

	rw, err := app.newResponseWriter(app.output)
	if err != nil {
		return err
	}

	rCh := make(chan *base.MultiResponse)

	wg := &sync.WaitGroup{}
	wg.Add(len(c.Devices))
	for n, d := range c.Devices {
		go app.runCommands(wg, n, d, rCh)

		resp := <-rCh
		go app.outputResult(wg, rw, n, d, resp)
	}

	wg.Wait()

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
