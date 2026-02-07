package main

import (
	"os"

	"github.com/bimmerbailey/cyro/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
