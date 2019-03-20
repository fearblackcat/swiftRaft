package raft

import (
	"context"
	"testing"
	"time"
)

func BenchmarkOneNode(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n := newNode()
	s := NewMemoryStorage()
	r := newTestRaft(1, []uint64{1}, 10, 1, s)
	go n.run(r)

	defer n.Stop()

	n.Campaign(ctx)
	go func() {
		for i := 0; i < b.N; i++ {
			n.Propose(ctx, []byte("foo"))
		}
	}()

	for {
		rd := <-n.Ready()
		s.Append(rd.Entries)
		// a reasonable disk sync latency
		time.Sleep(1 * time.Millisecond)
		n.Advance()
		if rd.HardState.Commit == uint64(b.N+1) {
			return
		}
	}
}
