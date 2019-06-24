package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	trstate "github.com/hashicorp/nomad/client/allocrunner/taskrunner/state"
	"github.com/hashicorp/nomad/client/state"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/plugins/base"
)

type ClientStateCommand struct {
}

func (a *ClientStateCommand) Help() string {
	helpText := `
Usage: nomad-debug client state <path_to_nomad_dir>

  Emits a json representation of the stored client state in json form.
`

	return strings.TrimSpace(helpText)
}

func (c *ClientStateCommand) Name() string { return "raft logs" }

func (c *ClientStateCommand) Synopsis() string {
	return "output content of client state"
}

func (c *ClientStateCommand) Run(args []string) int {
	if len(args) != 1 {
		return 1
	}

	logger := hclog.L()

	p := filepath.Join(args[0], "client")
	db, err := state.NewBoltStateDB(logger, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open client state: %v\n", err)
		return 1
	}
	defer db.Close()

	allocs, _, err := db.GetAllAllocations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get allocations: %v\n", err)
		return 1
	}

	data := map[string]*clientStateAlloc{}
	for _, alloc := range allocs {
		allocID := alloc.ID
		deployState, err := db.GetDeploymentStatus(allocID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get deployment status for %s: %v", allocID, err)
			return 1
		}

		tasks := map[string]*taskState{}
		tg := alloc.Job.LookupTaskGroup(alloc.TaskGroup)
		for _, jt := range tg.Tasks {
			ls, rs, err := db.GetTaskRunnerState(allocID, jt.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get task runner state %s: %v", allocID, err)
				return 1
			}

			var ds interface{}
			err = ls.TaskHandle.GetDriverState(&ds)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to parse driver state %s: %v", allocID, err)
				return 1
			}

			tasks[jt.Name] = &taskState{
				LocalState:  ls,
				RemoteState: rs,
				DriverState: ds,
			}
		}

		data[allocID] = &clientStateAlloc{
			Alloc:        alloc,
			DeployStatus: deployState,
			Tasks:        tasks,
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode output: %v\n", err)
		return 1
	}

	return 0
}

func unwrapDriverState(rawDriverConfig string) (interface{}, error) {
	if rawDriverConfig == "" {
		return nil, nil
	}

	b, err := base64.StdEncoding.DecodeString(rawDriverConfig)
	if err != nil {
		return "", err
	}
	var result interface{}
	err = base.MsgPackDecode(b, &result)
	if err != nil {
		return "", err
	}
	fixTime(result)

	return result, nil

}

type clientStateAlloc struct {
	Alloc        *structs.Allocation
	DeployStatus *structs.AllocDeploymentStatus
	Tasks        map[string]*taskState
}

type taskState struct {
	LocalState  *trstate.LocalState
	RemoteState *structs.TaskState
	DriverState interface{}
}
