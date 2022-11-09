package commando

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
	"gopkg.in/yaml.v2"
)

// Check for username/password set to 'prompt' instead of plaintext in inventory.yml
func (app *appCfg) setSecrets() {
	for name, cred := range app.credentials {
		if cred.Prompt == true {
			fmt.Printf("Credential name: `%s` is set to prompt:\n", name)
			fmt.Printf("Username: ")
			var user string
			_, err := fmt.Scanln(&user)
			if err != nil {
				log.Error("Error with username input.")
				os.Exit(1)
			}

			fmt.Printf("Password: ")
			bytepw, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Error("Error with password input.")
				os.Exit(1)
			}
			pass := string(bytepw)

			app.credentials[name].Username = user
			app.credentials[name].Password = pass

		}
	}
}

func (app *appCfg) loadInventoryFromYAML(i *inventory) error {
	yamlFile, err := os.ReadFile(app.inventory)
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
	app.setSecrets()

	// user-provided commands (via cli flag) take precedence over inventory
	if app.commands != "" {
		cmds := strings.Split(app.commands, "::")

		for _, device := range i.Devices {
			device.SendCommands = cmds
		}
	}

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

	app.credentials = map[string]*credentials{
		defaultName: {
			Username:          app.username,
			Password:          app.password,
			SecondaryPassword: app.password,
		},
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
		if !fRe.MatchString(n) {
			delete(i.Devices, n)
		}
	}
}
