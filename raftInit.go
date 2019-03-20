package smartRaft

import (
	"context"
	_ "net/http/pprof"
	"runtime/debug"

	"github.com/fearblackcat/smartRaft/node"
	"github.com/fearblackcat/smartRaft/raft/raftpb"
	"github.com/fearblackcat/smartRaft/raftsvr"
	"github.com/fearblackcat/smartRaft/utils/api/snap"

	"git.xiaojukeji.com/gulfstream/dcron/workflow/logtool"
)

func (a *Agent) setupRaft() {
	a.eventCh = make(chan event.Event)
	a.shutdownCh = make(chan struct{})
	a.proposeC = make(chan string)
	a.confCHangeC = make(chan raftpb.ConfChange)
	a.electedCh = make(chan bool)
	a.errCh = make(chan error)

	resmap, peers := a.genMemberList(a.config.Cluster)

	id := resmap[a.config.NodeName].id

	a.NodeID = id

	cfg := agentConfig{
		selfPeer:         a.config.AdvertiseRaftAddr,
		nodeName:         a.config.NodeName,
		join:             a.config.JoinCluster,
		proposeC:         a.proposeC,
		confChangeC:      a.confCHangeC,
		electedCh:        a.electedCh,
		errCh:            a.errCh,
		snapshotterReady: make(chan *snap.Snapshotter, 1),
		commitC:          make(chan *string),
		errorC:           make(chan error),
	}

	logtool.ZLog.Debug(logtool.DLTagUndefined, "gointo the raft setup")

	logtool.ZLog.Debug(logtool.DLTagUndefined, "new KV store")

	genSnapshot := func() ([]byte, error) { return a.kvs.GetSnapshot() }

	logtool.ZLog.Debug(logtool.DLTagUndefined, "ready to new raft node")

	node.NewRaftNode(id, peers, resmap, genSnapshot, &cfg)

	logtool.ZLog.Debug(logtool.DLTagUndefined, "ready to load data to map")

	a.kvs = raftsvr.NewKVStore(<-cfg.snapshotterReady, a.proposeC)

	go raftsvr.ServeHttpKVAPI(a.kvs, a.config.KvPort, a.confCHangeC, cfg.errorC)

	a.kvs.LoadDataToMap(cfg.commitC, cfg.errorC)

	logtool.ZLog.Debug(logtool.DLTagUndefined, "ready to serve http kv")

	logtool.ZLog.Debugf(context.TODO(), logtool.DLTagUndefined, "raftKvPort=%d", a.config.KvPort)
}

// Leader election routine
func (a *Agent) runForElection() {
	logtool.ZLog.Info(logtool.DLTagUndefined, "agent: Running for election")
	defer func() {
		if panicErr := recover(); panicErr != nil {
			logtool.ZLog.Errorf(context.TODO(), logtool.DLTagUndefined, "op=runForElection||panic=%s||stack=%s", panicErr, string(debug.Stack()))
		}
	}()

	for {
		select {
		case isElected := <-a.electedCh:
			if isElected {
				logtool.ZLog.Info(logtool.DLTagUndefined, "agent: Cluster leadership acquired")
			} else {
				logtool.ZLog.Info(logtool.DLTagUndefined, "agent: Cluster leadership lost")
			}

		case err := <-a.errCh:
			if err != nil {
				logtool.ZLog.Infof(context.TODO(), logtool.DLTagUndefined, "err=%s||eader election failed, channel is probably closed", err.Error())
			}
			return
		}
	}
}
