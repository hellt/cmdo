package commando

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

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
	fmt.Printf("     source: %s\n", "https://github.com/hellt/cmdo")
}
