package main

import (
	"encoding/json"
	"flag"
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

  Emit the nomad server state obtained by replaying the events of the raft log, in json format.

Options:

  --last-index=<last_index>
    Set the last log index to be applied, to drop spurious log entries not
    properly commited. If passed last_index is zero or negative, it's perceived
    as an offset from the last index seen in raft.
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
	var fLastIdx int64

	flags := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	flags.Usage = func() { fmt.Println(c.Help()) }
	flags.Int64Var(&fLastIdx, "last-index", 0, "")

	if err := flags.Parse(args); err != nil {
		return 1, fmt.Errorf("failed to parse arguments: %v", err)
	}
	args = flags.Args()

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

	// use dummy non-enabled FSM depedencies
	periodicDispatch := nomad.NewPeriodicDispatch(logger, nil)
	blockedEvals := nomad.NewBlockedEvals(nil, logger)
	evalBroker, err := nomad.NewEvalBroker(1, 1, 1, 1)
	if err != nil {
		return 1, err
	}
	fsmConfig := &nomad.FSMConfig{
		EvalBroker: evalBroker,
		Periodic:   periodicDispatch,
		Blocked:    blockedEvals,
		Logger:     logger,
		Region:     "default",
	}

	fsm, err := nomad.NewFSM(fsmConfig)
	if err != nil {
		return 1, err
	}
	lastIdx = lastIndex(lastIdx, fLastIdx)

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

func lastIndex(raftLastIdx uint64, cliLastIdx int64) uint64 {
	switch {
	case cliLastIdx < 0:
		if raftLastIdx > uint64(-cliLastIdx) {
			return raftLastIdx - uint64(-cliLastIdx)
		} else {
			return 0
		}
	case cliLastIdx == 0:
		return raftLastIdx
	case uint64(cliLastIdx) < raftLastIdx:
		return uint64(cliLastIdx)
	default:
		return raftLastIdx
	}
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
