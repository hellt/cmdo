package main

import (
	"os"

	cmdo "github.com/hellt/cmdo/app"
)

func main() {
	os.Exit(cmdo.CLI(os.Args[1:]))
}
