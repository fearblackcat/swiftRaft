package swiftRaft

import (
	"testing"
)

func TestRaftServer(t *testing.T) {
	cfg := &Config{
		Cluster:           "node01=http://127.0.0.1:12379",
		AdvertiseRaftAddr: "http://127.0.0.1:12379",
		NodeName:          "node01",
		JoinCluster:       false,
		KvPort:            9121,
		ElectedCh:         make(chan bool),
		ErrCh:             make(chan error),
	}

	NewRaftServer(cfg).Run()
}
