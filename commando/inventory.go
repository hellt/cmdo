package commando

import (
	"io/ioutil"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

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
