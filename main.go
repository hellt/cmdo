package main

import (
	"fmt"
	"io/ioutil"

	"sync"

	"github.com/fatih/color"
	"github.com/scrapli/scrapligo/driver/base"
	"github.com/scrapli/scrapligo/driver/core"
	"github.com/scrapli/scrapligo/transport"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Devices map[string]Device `yaml:"devices,omitempty"`
}

type Device struct {
	Platform     string   `yaml:"platform,omitempty"`
	Address      string   `yaml:"address,omitempty"`
	Username     string   `yaml:"username,omitempty"`
	Password     string   `yaml:"password,omitempty"`
	SendCommands []string `yaml:"send-commands,omitempty"`
}

func main() {
	// logging.SetDebugLogger(log.Print)
	c := &Config{}
	yamlFile, err := ioutil.ReadFile("inventory.yml")
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.UnmarshalStrict(yamlFile, c)
	if err != nil {
		log.Fatal(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(c.Devices))
	for n, d := range c.Devices {
		go runCommands(d, wg, n)
	}
	wg.Wait()

}

func runCommands(d Device, wg *sync.WaitGroup, name string) {
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
		log.Errorf("failed to create driver; error: %+v\n", err)
		return
	}

	err = driver.Open()
	if err != nil {
		log.Errorf("failed to open driver; error: %+v\n", err)
		return
	}

	r, err := driver.SendCommands(d.SendCommands)
	if err != nil {
		log.Errorf("failed to send commands; error: %+v\n", err)
		return
	}

	color.Green("\n**************************\n%s\n**************************\n", name)
	for idx, cmd := range d.SendCommands {
		c := color.New(color.Bold)
		c.Printf("\n-- %s:\n", cmd)
		fmt.Println(r.Responses[idx].Result)
	}

}
