package main

import (
	"os"

	"github.com/myorg/myapp/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
