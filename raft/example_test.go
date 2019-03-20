package raft

import (
	pb "github.com/fearblackcat/smartRaft/raft/raftpb"
)

func applyToStore(ents []pb.Entry)    {}
func sendMessages(msgs []pb.Message)  {}
func saveStateToDisk(st pb.HardState) {}
func saveToDisk(ents []pb.Entry)      {}

func ExampleNode() {
	c := &Config{}
	n := StartNode(c, nil)
	defer n.Stop()

	// stuff to n happens in other goroutines

	// the last known state
	var prev pb.HardState
	for {
		// Ready blocks until there is new state ready.
		rd := <-n.Ready()
		if !isHardStateEqual(prev, rd.HardState) {
			saveStateToDisk(rd.HardState)
			prev = rd.HardState
		}

		saveToDisk(rd.Entries)
		go applyToStore(rd.CommittedEntries)
		sendMessages(rd.Messages)
	}
}
