package main

import (
	"os"

	"github.com/ilyasturki/phasionary/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
