package commando

import (
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/scrapli/scrapligo/util"
	log "github.com/sirupsen/logrus"
)

func (app *appCfg) validTransport(t string) bool {
	switch t {
	case transport.SystemTransport:
		return true
	case transport.StandardTransport:
		return true
	case transport.TelnetTransport:
		return true
	default:
		return false
	}
}

func (app *appCfg) loadCredentials(o []util.Option, c string) ([]util.Option, error) {
	creds, ok := app.credentials[c]
	if !ok {
		return o, errInvalidCredentialsName
	}

	if creds.Username != "" {
		o = append(o, options.WithAuthUsername(creds.Username))
	}

	if creds.Password != "" {
		o = append(o, options.WithAuthPassword(creds.Password))
	}

	if creds.SecondaryPassword != "" {
		o = append(o, options.WithAuthSecondary(creds.SecondaryPassword))
	}

	if creds.PrivateKey != "" {
		o = append(o, options.WithAuthPrivateKey(creds.PrivateKey, ""))
	}

	return o, nil
}

func (app *appCfg) loadTransport(o []util.Option, t string) ([]util.Option, error) {
	// default to standard transport, so load those into options first
	o = append(
		o,
		options.WithTransportType(transport.StandardTransport),
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
		o = append(o, options.WithPort(transp.Port))
	}

	if !transp.StrictKey {
		o = append(o, options.WithAuthNoStrictKey())
	}

	if transp.SSHConfigFile != "" {
		o = append(o, options.WithSSHConfigFile(transp.SSHConfigFile))
	}

	if transp.TransportType != "" {
		if !app.validTransport(transp.TransportType) {
			return nil, errInvalidTransport
		}

		o = append(o, options.WithTransportType(transp.TransportType))
	}

	return o, nil
}

// loadOptions loads options from the provided inventory.
func (app *appCfg) loadOptions(d *device) ([]util.Option, error) {
	var o []util.Option

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

	plat, err := platform.NewPlatform(
		d.Platform,
		d.Address,
		o...,
	)
	if err != nil {
		log.Errorf("failed to create platform instance for device %s; error: %+v\n", err, name)
		return nil, err
	}

	driver, err = plat.GetNetworkDriver()
	if err != nil {
		log.Errorf("failed to create driver instance for device %s; error: %+v\n", err, name)
		return nil, err
	}

	err = driver.Open()
	if err != nil {
		log.Errorf("failed to open connection to device %s; error: %+v\n", err, name)

		return nil, err
	}

	return driver, nil
}
