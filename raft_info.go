package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RaftInfoCommand struct {
}

func (a *RaftInfoCommand) Help() string {
	helpText := `
Usage: nomad-debug info logs <path_to_nomad_dir>

  Emits some info about the raft logs.
`

	return strings.TrimSpace(helpText)
}

func (c *RaftInfoCommand) Name() string { return "raft info" }

func (c *RaftInfoCommand) Synopsis() string {
	return "output info of raft log"
}

func (c *RaftInfoCommand) Run(args []string) int {
	if len(args) != 1 {
		return 1
	}

	p := filepath.Join(args[0], "server", "raft", "raft.db")

	store, firstIdx, lastIdx, err := raftState(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open raft logs: %v\n", err)
		return 1
	}
	defer store.Close()

	fmt.Println("path:        ", p)
	fmt.Println("length:      ", lastIdx-firstIdx+1)
	fmt.Println("first index: ", firstIdx)
	fmt.Println("last index:  ", lastIdx)

	return 0
}
