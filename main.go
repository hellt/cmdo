package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/glspi/cmdo/commando"
)

func main() {
	err := commando.NewCLI().Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
