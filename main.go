package main

import (
	"os"

	"github.com/hellt/cmdo/cmdo"
)

func main() {
	os.Exit(cmdo.CLI(os.Args[1:]))
}
