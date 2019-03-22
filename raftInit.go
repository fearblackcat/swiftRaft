package swiftRaft

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/fearblackcat/swiftRaft/node"
	"github.com/fearblackcat/swiftRaft/raft/raftpb"
	"github.com/fearblackcat/swiftRaft/raftsvr"
	"github.com/fearblackcat/swiftRaft/utils/api/snap"
	"github.com/fearblackcat/swiftRaft/utils/logtool"
)

type Config struct {
	Cluster           string
	AdvertiseRaftAddr string
	NodeName          string
	JoinCluster       bool
	KvPort            int
	ElectedCh         chan bool
	ErrCh             chan error
}

type RaftServer struct {
	cfg         *Config
	nodeID      uint64
	electedCh   chan bool
	errCh       chan error
	shutdownCh  chan struct{}
	proposeC    chan string
	confCHangeC chan raftpb.ConfChange
	kvs         *raftsvr.Kvstore
}

func NewRaftServer(cfg *Config) *RaftServer {
	if cfg == nil || cfg.ElectedCh == nil ||
		cfg.ErrCh == nil || len(cfg.Cluster) == 0 ||
		len(cfg.AdvertiseRaftAddr) == 0 ||
		len(cfg.NodeName) == 0 ||
		cfg.KvPort == 0 {
		fmt.Fprint(os.Stderr, "manditory fields of configuration is empty \n")
		return nil
	}

	r := &RaftServer{}

	r.cfg = cfg
	r.electedCh = make(chan bool)
	r.errCh = make(chan error)
	r.shutdownCh = make(chan struct{})
	r.proposeC = make(chan string)
	r.confCHangeC = make(chan raftpb.ConfChange)

	go r.setupRaft()

	return r
}

func (r *RaftServer) genMemberList(cluster string) (map[string]node.MemberInfo, []string) {
	var peers []string

	resmap := make(map[string]node.MemberInfo)
	kvs := strings.Split(cluster, ",")
	for _, v := range kvs {
		kv := strings.Split(v, "=")
		peers = append(peers, kv[1])
	}
	for _, v := range kvs {
		kv := strings.Split(v, "=")
		var b []byte
		sort.Strings(peers)
		for _, p := range peers {
			b = append(b, []byte(p)...)
		}

		b = append(b, []byte(kv[0])...)

		hash := sha1.Sum(b)
		id := binary.BigEndian.Uint64(hash[:8])
		resmap[kv[0]] = node.MemberInfo{
			ID:   id,
			Peer: kv[1],
		}
	}

	return resmap, peers
}

func (r *RaftServer) setupRaft() {
	if r == nil {
		return
	}

	resMap, peers := r.genMemberList(r.cfg.Cluster)

	id := resMap[r.cfg.NodeName].ID

	r.nodeID = id

	cfg := node.RaftConfig{
		SelfPeer:      r.cfg.AdvertiseRaftAddr,
		NodeName:      r.cfg.NodeName,
		Join:          r.cfg.JoinCluster,
		ProposeC:      r.proposeC,
		ConfChangeC:   r.confCHangeC,
		ElectedCh:     r.electedCh,
		ErrCh:         r.errCh,
		SnapshotReady: make(chan *snap.Snapshotter, 1),
		CommitC:       make(chan *string),
		ErrorC:        make(chan error),
	}

	logtool.NLog.Debug("gointo the raft setup")

	logtool.NLog.Debug("new KV store")

	genSnapshot := func() ([]byte, error) { return r.kvs.GetSnapshot() }

	logtool.NLog.Debug("ready to new raft node")

	node.NewRaftNode(id, peers, resMap, genSnapshot, &cfg)

	logtool.NLog.Debug("ready to load data to map")

	r.kvs = raftsvr.NewKVStore(<-cfg.SnapshotReady, r.proposeC)

	go raftsvr.ServeHttpKVAPI(r.kvs, r.cfg.KvPort, r.confCHangeC, cfg.ErrorC)

	r.kvs.LoadDataToMap(cfg.CommitC, cfg.ErrorC)

	logtool.NLog.Debug("ready to serve http kv")

	logtool.NLog.Debugf("raftKvPort=%d", r.cfg.KvPort)
}

// Leader election routine
func (r *RaftServer) Run() {
	if r == nil {
		return
	}
	logtool.NLog.Info("agent: Running for election")
	defer func() {
		if panicErr := recover(); panicErr != nil {
			logtool.NLog.Errorf("op=runForElection||panic=%s||stack=%s", panicErr, string(debug.Stack()))
		}
	}()

	for {
		select {
		case isElected := <-r.electedCh:
			if isElected {
				logtool.NLog.Info("agent: Cluster leadership acquired")
			} else {
				logtool.NLog.Info("agent: Cluster leadership lost")
			}
			go func() {
				r.cfg.ElectedCh <- isElected
			}()

		case err := <-r.errCh:
			if err != nil {
				logtool.NLog.Infof("err=%s||eader election failed, channel is probably closed", err.Error())
			}
			go func() {
				r.cfg.ErrCh <- err
			}()

			return
		}
	}
}
