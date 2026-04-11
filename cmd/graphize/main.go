// Package main provides the graphize CLI.
package main

import (
	"os"

	"github.com/plexusone/graphize/cmd/graphize/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
