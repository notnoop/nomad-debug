# Debug nomad state

Have you needed to inspect nomad state?  Investigate how a job/alloc got to its state?  Investigate behavior of client on restore?  Research nomad server Raft FSM changes by replaying raft transactions and inspecting state?  If so, you are in luck!  `nomad-debug` helps debugging such scenarios by dumping the state of nomad in json format, for further analysis by `jq`.

The main commands are:

```
# dump all raft log entries as json array to stdout
nomad-debug raft logs <nomad-data-dir>

# dump the nomad server state store, by replaying raft log events
nomad-debug raft state <nomad-data-dir>

# dump the nomad client state
nomad-debug client state <nomad-data-dir>
```

## Caveats

* The raft logs may not represent cluster state accurately at time of server shutting down.  The raft log main contain:
  * some uncommitted log entries: these are transactions that haven't fully replicated to quorum of nodes, so actual state may lag behind the state found here
  * some spurious log entries: specially around leader election, some persisted logs might be some garbage to be overwriten later
  * some missing log entries: the raft logs of a follower might be lagging behind the leader

* `client state` only works against Nomad 0.9 client.  Client 0.8 and earlier are not supported.

## How to use

Checkout this repository as a subdir of `nomad` and run `go install`:

```
$ cd ~/go/src/github.com/hashicorp/nomad/
$ git clone git@github.com:notnoop/nomad-debug.git
$ cd nomad-debug
$ go install .
```

## TODO

* [ ] Support nomad server raft snapshoted state
* [ ] Export to a database (e.g. sqlite, postgresql) to ease querying against database
* [ ] Vendor nomad and its dependencies to avoid needing to checkout as a subdirectory
