package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/nomad/version"
)

type VersionCommand struct{}

func (a *VersionCommand) Help() string {
	helpText := `
Usage: nomad-debug version

  Emits version infromation about the nomad-debug command.
`

	return strings.TrimSpace(helpText)
}

func (a *VersionCommand) Name() string { return "version" }

func (a *VersionCommand) Synopsis() string {
	return "output nomad-debug version"
}

func (a *VersionCommand) Run(args []string) int {
	fmt.Printf("nomad-debug built with %s\n", version.GetVersion().FullVersionNumber(true))
	return 0
}
