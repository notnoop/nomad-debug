package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"
)

func main() {
	commands := map[string]cli.CommandFactory{
		"raft info": func() (cli.Command, error) {
			return &RaftInfoCommand{}, nil
		},
		"raft logs": func() (cli.Command, error) {
			return &RaftLogsCommand{}, nil
		},
		"raft state": func() (cli.Command, error) {
			return &RaftStateCommand{}, nil
		},
		"client state": func() (cli.Command, error) {
			return &ClientStateCommand{}, nil
		},
	}
	cli := &cli.CLI{
		Name:       "nomad-debug",
		Args:       os.Args[1:],
		HelpWriter: os.Stdout,
		Commands:   commands,
	}

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
	}

	os.Exit(exitCode)
}
