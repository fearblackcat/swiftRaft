// Package v2stats defines a standard interface for etcd cluster statistics.
package v2stats

import "github.com/coreos/pkg/capnslog"

var plog = capnslog.NewPackageLogger("github.com/fearblackcat/swiftRaft", "utils/api/v2stats")

type Stats interface {
	// SelfStats returns the struct representing statistics of this server
	SelfStats() []byte
	// LeaderStats returns the statistics of all followers in the cluster
	// if this server is leader. Otherwise, nil is returned.
	LeaderStats() []byte
	// StoreStats returns statistics of the store backing this EtcdServer
	StoreStats() []byte
}
