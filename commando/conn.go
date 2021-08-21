package commando

import (
	"github.com/scrapli/scrapligo/driver/base"
	"github.com/scrapli/scrapligo/driver/core"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/transport"
	log "github.com/sirupsen/logrus"
	"github.com/srl-labs/srlinux-scrapli"
)

func (app *appCfg) validTransport(t string) bool {
	switch t {
	case transport.SystemTransportName:
		return true
	case transport.StandardTransportName:
		return true
	case transport.TelnetTransportName:
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
	o = append(
		o,
		base.WithTransportType(transport.StandardTransportName),
		base.WithAuthStrictKey(false),
	)

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

func (app *appCfg) openCoreConn(name string, d *device) (*network.Driver, error) {
	var driver *network.Driver

	o, err := app.loadOptions(d)
	if err != nil {
		log.Errorf(
			"failed to load credentials or transport options for %s; error: %+v\n",
			name,
			err,
		)

		return nil, err
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
		return nil, err
	}

	err = driver.Open()
	if err != nil {
		log.Errorf("failed to open connection to device %s; error: %+v\n", err, name)

		return nil, err
	}

	return driver, nil
}
