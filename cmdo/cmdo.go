package cmdo

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"sync"

	"github.com/fatih/color"
	"github.com/scrapli/scrapligo/driver/base"
	"github.com/scrapli/scrapligo/driver/core"
	"github.com/scrapli/scrapligo/transport"
	log "github.com/sirupsen/logrus"
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

type appCfg struct {
	inventory string // path to inventory file
	output    string // output mode
}

func (app *appCfg) fromArgs(args []string) error {
	fl := flag.NewFlagSet("cmdo", flag.ContinueOnError)
	fl.StringVar(&app.inventory, "i", "inventory.yml", "path to the inventory file")
	fl.StringVar(&app.output, "o", "file", "print output to: [file, stdout]")
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

	rCh := make(chan *base.MultiResponse)

	wg := &sync.WaitGroup{}
	wg.Add(len(c.Devices))
	for n, d := range c.Devices {
		go runCommands(wg, n, d, rCh)
	}

	for n, d := range c.Devices {
		resp := <-rCh
		wg.Add(1)
		go outputResult(wg, n, d, resp)
	}

	wg.Wait()

	return nil
}

func runCommands(wg *sync.WaitGroup, name string, d device, rCh chan<- *base.MultiResponse) {
	defer wg.Done()
	driver, err := core.NewCoreDriver(
		d.Address,
		d.Platform,
		base.WithAuthStrictKey(false),
		base.WithAuthUsername(d.Username),
		base.WithAuthPassword(d.Password),
		base.WithTransportType(transport.StandardTransportName),
	)

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

func outputResult(wg *sync.WaitGroup, name string, d device, r *base.MultiResponse) {
	defer wg.Done()
	color.Green("\n**************************\n%s\n**************************\n", name)
	for idx, cmd := range d.SendCommands {
		c := color.New(color.Bold)
		c.Printf("\n-- %s:\n", cmd)
		fmt.Println(r.Responses[idx].Result)
	}
}
