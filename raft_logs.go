package main

//go:generate ./generate_msgtypes.sh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-msgpack/codec"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type RaftLogsCommand struct {
}

func (a *RaftLogsCommand) Help() string {
	helpText := `
Usage: nomad-debug raft logs <path_to_nomad_dir>

  Emits the raft logs content in json form.
`

	return strings.TrimSpace(helpText)
}

func (c *RaftLogsCommand) Name() string { return "raft logs" }

func (c *RaftLogsCommand) Synopsis() string {
	return "output content of raft log"
}

func (c *RaftLogsCommand) Run(args []string) int {
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

	arr := make([]*logMessage, 0, lastIdx-firstIdx+1)
	for i := firstIdx; i <= lastIdx; i++ {
		var e raft.Log
		err := store.GetLog(i, &e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read log entry at index %d: %v\n", i, err)
			continue
			//return 1
		}

		m, err := decode(&e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to decode log entry at index %d: %v\n", i, err)
			continue
			//return 1
		}

		arr = append(arr, m)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(arr); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode output: %v\n", err)
		return 1
	}

	return 0
}

func raftState(p string) (store *raftboltdb.BoltStore, firstIdx uint64, lastIdx uint64, err error) {
	s, err := raftboltdb.NewBoltStore(p)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to open raft logs: %v", err)
	}

	firstIdx, err = s.FirstIndex()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to fetch first index: %v", err)
	}

	lastIdx, err = s.LastIndex()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to fetch last index: %v", err)
	}

	return s, firstIdx, lastIdx, nil
}

type logMessage struct {
	LogType string
	Term    uint64
	Index   uint64

	CommandType           string      `json:",omitempty"`
	IgnoreUnknownTypeFlag bool        `json:",omitempty"`
	Body                  interface{} `json:",omitempty"`
}

func decode(e *raft.Log) (*logMessage, error) {
	m := &logMessage{
		LogType: logTypes[e.Type],
		Term:    e.Term,
		Index:   e.Index,
	}

	if m.LogType == "" {
		m.LogType = fmt.Sprintf("%d", e.Type)
	}

	var data []byte
	if e.Type == raft.LogCommand {
		if len(e.Data) == 0 {
			return nil, fmt.Errorf("command did not include data")
		}

		msgType := structs.MessageType(e.Data[0])

		m.CommandType = msgTypeNames[msgType & ^structs.IgnoreUnknownTypeFlag]
		m.IgnoreUnknownTypeFlag = (msgType & structs.IgnoreUnknownTypeFlag) != 0

		data = e.Data[1:]
	} else {
		data = e.Data
	}

	if len(data) != 0 {
		decoder := codec.NewDecoder(bytes.NewReader(data), MsgpackHandle)

		var v interface{}
		var err error
		if m.CommandType == msgTypeNames[structs.JobBatchDeregisterRequestType] {
			var vr structs.JobBatchDeregisterRequest
			err = decoder.Decode(&vr)
			v = jsonifyJobBatchDeregisterRequest(&vr)
		} else {
			var vr interface{}
			err = decoder.Decode(&vr)
			v = vr
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to decode log entry at index %d: failed to decode body of %v.%v %v\n", e.Index, e.Type, m.CommandType, err)
			v = "FAILED TO DECODE DATA"
		}
		fixTime(v)
		m.Body = v
	}

	return m, nil
}

func jsonifyJobBatchDeregisterRequest(v *structs.JobBatchDeregisterRequest) interface{} {
	var data struct {
		Jobs  map[string]*structs.JobDeregisterOptions
		Evals []*structs.Evaluation
		structs.WriteRequest
	}
	data.Evals = v.Evals
	data.WriteRequest = v.WriteRequest

	data.Jobs = make(map[string]*structs.JobDeregisterOptions, len(v.Jobs))
	if len(v.Jobs) != 0 {
		for k, v := range v.Jobs {
			data.Jobs[k.Namespace+"."+k.ID] = v
		}
	}
	return data
}

var logTypes = map[raft.LogType]string{
	raft.LogCommand:              "LogCommand",
	raft.LogNoop:                 "LogNoop",
	raft.LogAddPeerDeprecated:    "LogAddPeerDeprecated",
	raft.LogRemovePeerDeprecated: "LogRemovePeerDeprecated",
	raft.LogBarrier:              "LogBarrier",
	raft.LogConfiguration:        "LogConfiguration",
}
