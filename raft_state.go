package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/nomad/nomad"
	"github.com/hashicorp/raft"
)

type RaftStateCommand struct {
}

func (a *RaftStateCommand) Help() string {
	helpText := `
Usage: nomad-debug raft state <path_to_nomad_dir>

  Emits the stored state represented by the raft logs, in json form.
`

	return strings.TrimSpace(helpText)
}

func (c *RaftStateCommand) Name() string { return "raft logs" }

func (c *RaftStateCommand) Synopsis() string {
	return "output content of raft log"
}

func (c *RaftStateCommand) Run(args []string) int {
	r, err := c.run(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return r
}

func (c *RaftStateCommand) run(args []string) (int, error) {
	if len(args) != 1 {
		return 1, fmt.Errorf("expected one arg but got %d", len(args))
	}

	p := filepath.Join(args[0], "server", "raft", "raft.db")

	store, firstIdx, lastIdx, err := raftState(p)
	if err != nil {
		return 1, fmt.Errorf("failed to open raft logs: %v", err)
	}
	defer store.Close()

	logger := hclog.L()
	// Create an eval broker
	evalBroker, err := nomad.NewEvalBroker(1, 1, 1, 1)
	if err != nil {
		return 1, err
	}
	fsmConfig := &nomad.FSMConfig{
		EvalBroker: evalBroker,
		Periodic:   nomad.NewPeriodicDispatch(logger, nil),
		Blocked:    nomad.NewBlockedEvals(nil, hclog.L()),
		Logger:     hclog.L(),
		Region:     "default",
	}

	fsm, err := nomad.NewFSM(fsmConfig)
	if err != nil {
		return 1, err
	}

	for i := firstIdx; i <= lastIdx; i++ {
		var e raft.Log
		err := store.GetLog(i, &e)
		if err != nil {
			return 1, fmt.Errorf("failed to read log entry at index %d: %v", i, err)
		}

		if e.Type == raft.LogCommand {
			fsm.Apply(&e)
		}
	}

	state := fsm.State()
	result := map[string][]interface{}{
		"ACLPolicies":      toArray(state.ACLPolicies(nil)),
		"ACLTokens":        toArray(state.ACLTokens(nil)),
		"Allocs":           toArray(state.Allocs(nil)),
		"Deployments":      toArray(state.Deployments(nil)),
		"Evals":            toArray(state.Evals(nil)),
		"Indexes":          toArray(state.Indexes()),
		"JobSummaries":     toArray(state.JobSummaries(nil)),
		"JobVersions":      toArray(state.JobVersions(nil)),
		"Jobs":             toArray(state.Jobs(nil)),
		"Nodes":            toArray(state.Nodes(nil)),
		"PeriodicLaunches": toArray(state.PeriodicLaunches(nil)),
		"VaultAccessors":   toArray(state.VaultAccessors(nil)),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return 1, fmt.Errorf("failed to encode output: %v", err)
	}

	return 0, nil
}

func toArray(iter memdb.ResultIterator, err error) []interface{} {
	if err != nil {
		return []interface{}{err}
	}

	r := []interface{}{}

	item := iter.Next()
	for item != nil {
		r = append(r, item)
		item = iter.Next()
	}

	return r
}
