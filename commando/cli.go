package commando

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// NewCLI defines the CLI flags and commands.
func NewCLI() *cli.App {
	appC := &appCfg{}
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "inventory",
			Aliases:     []string{"i"},
			Value:       "inventory.yml",
			Usage:       "path to the inventory file",
			Destination: &appC.inventory,
		},
		&cli.StringFlag{
			Name:        "output",
			Aliases:     []string{"o"},
			Value:       "file",
			Usage:       "output destination. One of: [file, stdout]",
			Destination: &appC.output,
		},
		&cli.BoolFlag{
			Name:        "add-timestamp",
			Aliases:     []string{"t"},
			Value:       false,
			Usage:       "append timestamp to output directory",
			Destination: &appC.timestamp,
		},
		&cli.StringFlag{
			Name:        "filter",
			Aliases:     []string{"f"},
			Value:       "",
			Usage:       "filter to select the devices to send commands to",
			Destination: &appC.devFilter,
		},
		&cli.StringFlag{
			Name:        "platform",
			Aliases:     []string{"k"},
			Value:       "",
			Usage:       "platform name [only for single-node mode]",
			Destination: &appC.platform,
		},
		&cli.StringFlag{
			Name:        "address",
			Aliases:     []string{"a"},
			Value:       "",
			Usage:       "device's address [only for single-node mode]",
			Destination: &appC.address,
		},
		&cli.StringFlag{
			Name:        "username",
			Aliases:     []string{"u"},
			Value:       "",
			Usage:       "username to use for SSH connection",
			Destination: &appC.username,
		},
		&cli.StringFlag{
			Name:        "password",
			Aliases:     []string{"p"},
			Value:       "",
			Usage:       "username to use for SSH connection",
			Destination: &appC.password,
		},
		&cli.StringFlag{
			Name:        "commands",
			Aliases:     []string{"c"},
			Usage:       "commands to send. separated with ::",
			Destination: &appC.commands,
		},
	}

	cli.VersionPrinter = showVersion

	app := &cli.App{
		Name:    "cmdo",
		Version: "dev",
		Usage:   "run commands against network devices",
		Flags:   flags,
		Action: func(c *cli.Context) error {
			return appC.run()
		},
	}

	return app
}

func showVersion(c *cli.Context) {
	fmt.Printf("    version: %s\n", version)
	fmt.Printf("     commit: %s\n", commit)
	fmt.Printf("     source: %s\n", "https://github.com/glspi/cmdo")
}
