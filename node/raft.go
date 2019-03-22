package node

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/fearblackcat/swiftRaft/raft"
	"github.com/fearblackcat/swiftRaft/raft/raftpb"
	"github.com/fearblackcat/swiftRaft/raftsvr"
	"github.com/fearblackcat/swiftRaft/utils/api/rafthttp"
	"github.com/fearblackcat/swiftRaft/utils/api/snap"
	stats "github.com/fearblackcat/swiftRaft/utils/api/v2stats"
	"github.com/fearblackcat/swiftRaft/utils/api/wal"
	"github.com/fearblackcat/swiftRaft/utils/api/wal/walpb"
	"github.com/fearblackcat/swiftRaft/utils/errhandle"
	"github.com/fearblackcat/swiftRaft/utils/logtool"
	"github.com/fearblackcat/swiftRaft/utils/pkg/fileutil"
	"github.com/fearblackcat/swiftRaft/utils/pkg/types"
)

type MemberInfo struct {
	ID   uint64
	Peer string
}

// A key-value stream backed by raft
type raftNode struct {
	proposeC    <-chan string            // proposed messages (k,v)
	confChangeC <-chan raftpb.ConfChange // proposed cluster config changes
	commitC     chan<- *string           // entries committed to log (k,v)
	errorC      chan<- error             // errors from raft session

	nodeName    string
	selfPeer    string
	id          uint64   // client ID for raft session
	peers       []string // raft peer URLs
	members     map[string]MemberInfo
	join        bool   // node is joining an existing cluster
	waldir      string // path to WAL directory
	snapdir     string // path to snapshot directory
	getSnapshot func() ([]byte, error)
	lastIndex   uint64 // index of log at start

	confState     raftpb.ConfState
	snapshotIndex uint64
	appliedIndex  uint64

	// raft backing for the commit/error channel
	node        raft.Node
	raftStorage *raft.MemoryStorage
	wal         *wal.WAL

	snapshotter      *snap.Snapshotter
	snapshotterReady chan *snap.Snapshotter // signals when snapshotter is ready

	snapCount uint64
	transport *rafthttp.Transport
	stopc     chan struct{} // signals proposal channel closed
	httpstopc chan struct{} // signals http server to shutdown
	httpdonec chan struct{} // signals http server shutdown complete
}

type RaftConfig struct {
	SelfPeer      string
	NodeName      string
	Join          bool
	ProposeC      <-chan string
	ConfChangeC   <-chan raftpb.ConfChange
	ElectedCh     chan bool
	ErrCh         chan error
	SnapshotReady chan *snap.Snapshotter
	CommitC       chan *string
	ErrorC        chan error
}

var defaultSnapshotCount uint64 = 10000

// newRaftNode initiates a raft instance and returns a committed log entry
// channel and error channel. Proposals for log updates are sent over the
// provided the proposal channel. All log entries are replayed over the
// commit channel, followed by a nil message (to indicate the channel is
// current), then new log entries. To shutdown, close proposeC and read errorC.
func NewRaftNode(id uint64, peers []string, members map[string]MemberInfo, getSnapshot func() ([]byte, error), cfg *RaftConfig) {

	rc := &raftNode{
		proposeC:    cfg.ProposeC,
		confChangeC: cfg.ConfChangeC,
		commitC:     cfg.CommitC,
		errorC:      cfg.ErrorC,
		id:          id,
		selfPeer:    cfg.SelfPeer,
		nodeName:    cfg.NodeName,
		peers:       peers,
		members:     members,
		join:        cfg.Join,
		waldir:      fmt.Sprintf("raft-%s", cfg.NodeName),
		snapdir:     fmt.Sprintf("raft-%s-snap", cfg.NodeName),
		getSnapshot: getSnapshot,
		snapCount:   defaultSnapshotCount,
		stopc:       make(chan struct{}),
		httpstopc:   make(chan struct{}),
		httpdonec:   make(chan struct{}),

		snapshotterReady: cfg.SnapshotReady,
		// rest of structure populated after WAL replay
	}
	go rc.startRaft(cfg.ElectedCh, cfg.ErrCh)
}

func (rc *raftNode) saveSnap(snap raftpb.Snapshot) error {
	// must save the snapshot index to the WAL before saving the
	// snapshot to maintain the invariant that we only Open the
	// wal at previously-saved snapshot indexes.
	walSnap := walpb.Snapshot{
		Index: snap.Metadata.Index,
		Term:  snap.Metadata.Term,
	}
	if err := rc.wal.SaveSnapshot(walSnap); err != nil {
		return err
	}
	if err := rc.snapshotter.SaveSnap(snap); err != nil {
		return err
	}
	return rc.wal.ReleaseLockTo(snap.Metadata.Index)
}

func (rc *raftNode) entriesToApply(ents []raftpb.Entry) (nents []raftpb.Entry) {
	if len(ents) == 0 {
		return
	}
	firstIdx := ents[0].Index
	if firstIdx > rc.appliedIndex+1 {
		logtool.RLog.Fatal("first index of committed entry should <= progress.appliedIndex+1", map[string]interface{}{
			"entry":                 firstIdx,
			"progress.appliedIndex": rc.appliedIndex,
		})
	}
	if rc.appliedIndex-firstIdx+1 < uint64(len(ents)) {
		nents = ents[rc.appliedIndex-firstIdx+1:]
	}
	return nents
}

// publishEntries writes committed log entries to commit channel and returns
// whether all entries could be published.
func (rc *raftNode) publishEntries(ents []raftpb.Entry) bool {
	for i := range ents {
		switch ents[i].Type {
		case raftpb.EntryNormal:
			if len(ents[i].Data) == 0 {
				// ignore empty messages
				break
			}
			s := string(ents[i].Data)
			select {
			case rc.commitC <- &s:
			case <-rc.stopc:
				return false
			}

		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			cc.Unmarshal(ents[i].Data)
			rc.confState = *rc.node.ApplyConfChange(cc)
			switch cc.Type {
			case raftpb.ConfChangeAddNode:
				if len(cc.Context) > 0 {
					rc.transport.AddPeer(types.ID(cc.NodeID), []string{string(cc.Context)})
				}
			case raftpb.ConfChangeRemoveNode:
				if cc.NodeID == uint64(rc.id) {
					logtool.RLog.Info("I've been removed from the cluster! Shutting down.", map[string]interface{}{})
					return false
				}
				rc.transport.RemovePeer(types.ID(cc.NodeID))
			}
		}

		// after commit, update appliedIndex
		rc.appliedIndex = ents[i].Index

		// special nil commit to signal replay has finished
		if ents[i].Index == rc.lastIndex {
			select {
			case rc.commitC <- nil:
			case <-rc.stopc:
				return false
			}
		}
	}
	return true
}

func (rc *raftNode) loadSnapshot() *raftpb.Snapshot {
	snapshot, err := rc.snapshotter.Load()
	if err != nil && err != snap.ErrNoSnapshot {
		logtool.RLog.Fatal("raft: error loading snapshot", map[string]interface{}{
			"error": err,
		})
	}
	return snapshot
}

// openWAL returns a WAL ready for reading.
func (rc *raftNode) openWAL(snapshot *raftpb.Snapshot) *wal.WAL {
	if !wal.Exist(rc.waldir) {
		if err := os.Mkdir(rc.waldir, 0750); err != nil {
			logtool.RLog.Fatal("raft: cannot create dir for wal ", map[string]interface{}{
				"error": err,
			})
		}

		w, err := wal.Create(logtool.RLog, rc.waldir, nil)
		if err != nil {
			logtool.RLog.Fatal("raft: create wal error ", map[string]interface{}{
				"error": err,
			})
		}
		w.Close()
	}

	walsnap := walpb.Snapshot{}
	if snapshot != nil {
		walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
	}
	logtool.RLog.Info("loading WAL at term and index ", map[string]interface{}{
		"term":  walsnap.Term,
		"index": walsnap.Index,
	})
	w, err := wal.Open(logtool.RLog, rc.waldir, walsnap)
	if err != nil {
		logtool.RLog.Fatal("raft: error loading wal (%v)", map[string]interface{}{
			"error": err,
		})
	}

	return w
}

// replayWAL replays WAL entries into the raft instance.
func (rc *raftNode) replayWAL() *wal.WAL {
	logtool.RLog.Info("replaying WAL of member", map[string]interface{}{
		"member id": rc.id,
	})
	snapshot := rc.loadSnapshot()
	w := rc.openWAL(snapshot)
	_, st, ents, err := w.ReadAll()
	if err != nil {
		logtool.RLog.Fatal("raft: failed to read WAL ", map[string]interface{}{
			"error": err,
		})
	}
	rc.raftStorage = raft.NewMemoryStorage()
	if snapshot != nil {
		rc.raftStorage.ApplySnapshot(*snapshot)
	}
	rc.raftStorage.SetHardState(st)

	// append to storage so raft starts at the right place in log
	rc.raftStorage.Append(ents)
	// send nil once lastIndex is published so client knows commit channel is current
	if len(ents) > 0 {
		rc.lastIndex = ents[len(ents)-1].Index
	} else {
		rc.commitC <- nil
	}
	return w
}

func (rc *raftNode) writeError(err error) {
	rc.stopHTTP()
	close(rc.commitC)
	rc.errorC <- err
	close(rc.errorC)
	rc.node.Stop()
}

func (rc *raftNode) startRaft(electedCh chan bool, errCh chan error) {
	if !fileutil.Exist(rc.snapdir) {
		if err := os.Mkdir(rc.snapdir, 0750); err != nil {
			logtool.RLog.Fatal("raft: cannot create dir for snapshot", map[string]interface{}{
				"error": err,
			})
		}
	}
	rc.snapshotter = snap.New(logtool.RLog, rc.snapdir)
	rc.snapshotterReady <- rc.snapshotter

	hostname, err := os.Hostname()
	if err != nil {
		logtool.NLog.Fatal(err.Error())
	}
	logtool.InitRaftLogger("debug", hostname)

	logtool.InitNodeMsgLogger("debug", hostname)

	oldwal := wal.Exist(rc.waldir)
	rc.wal = rc.replayWAL()

	rpeers := make([]raft.Peer, len(rc.peers))
	var i = 0
	for _, v := range rc.members {
		rpeers[i] = raft.Peer{ID: v.ID}
		i++
	}
	c := &raft.Config{
		ID:                        uint64(rc.id),
		ElectionTick:              10,
		HeartbeatTick:             1,
		Storage:                   rc.raftStorage,
		MaxSizePerMsg:             1024 * 1024,
		MaxInflightMsgs:           256,
		MaxUncommittedEntriesSize: 1 << 30,
		Logger: logtool.NLog,
	}

	if oldwal {
		rc.node = raft.RestartNode(c)
	} else {
		startPeers := rpeers
		if rc.join {
			startPeers = nil
		}
		rc.node = raft.StartNode(c, startPeers)
	}

	rc.transport = &rafthttp.Transport{
		Logger:      logtool.RLog,
		ID:          types.ID(rc.id),
		ClusterID:   0x1000,
		Raft:        rc,
		ServerStats: stats.NewServerStats("", ""),
		LeaderStats: stats.NewLeaderStats(strconv.FormatUint(rc.id, 10)),
		ErrorC:      make(chan error),
	}

	rc.transport.Start()
	for k, v := range rc.members {
		if k != rc.nodeName {
			rc.transport.AddPeer(types.ID(v.ID), []string{v.Peer})
		}
	}

	rsvr := raftsvr.NewServerAttach(rc.waldir, rc.snapdir, rc.stopc, rc.httpdonec)

	go rc.serveRaft()
	go rc.serveChannels(electedCh, errCh)
	go rsvr.GoAttach(rsvr.PurgeFile)
}

// stop closes http, closes all channels, and stops raft.
func (rc *raftNode) stop() {
	rc.stopHTTP()
	close(rc.commitC)
	close(rc.errorC)
	rc.node.Stop()
}

func (rc *raftNode) stopHTTP() {
	rc.transport.Stop()
	close(rc.httpstopc)
	<-rc.httpdonec
}

func (rc *raftNode) publishSnapshot(snapshotToSave raftpb.Snapshot) {
	if raft.IsEmptySnap(snapshotToSave) {
		return
	}

	logtool.RLog.Info("publishing snapshot at index", map[string]interface{}{
		"index": rc.snapshotIndex,
	})

	defer logtool.RLog.Info("finished publishing snapshot at index", map[string]interface{}{
		"index": rc.snapshotIndex,
	})

	if snapshotToSave.Metadata.Index <= rc.appliedIndex {
		logtool.RLog.Fatal("snapshot index should > progress.appliedIndex ", map[string]interface{}{
			"index":                 snapshotToSave.Metadata.Index,
			"progress.appliedIndex": rc.appliedIndex,
		})
	}
	rc.commitC <- nil // trigger kvstore to load snapshot

	rc.confState = snapshotToSave.Metadata.ConfState
	rc.snapshotIndex = snapshotToSave.Metadata.Index
	rc.appliedIndex = snapshotToSave.Metadata.Index
}

var snapshotCatchUpEntriesN uint64 = 10000

func (rc *raftNode) maybeTriggerSnapshot() {
	if rc.appliedIndex-rc.snapshotIndex <= rc.snapCount {
		return
	}

	logtool.RLog.Info("start snapshot [applied index | last snapshot index]", map[string]interface{}{
		"applied index":  rc.appliedIndex,
		"snapshot index": rc.snapshotIndex,
	})
	data, err := rc.getSnapshot()
	if err != nil {
		log.Panic(err)
	}
	snap, err := rc.raftStorage.CreateSnapshot(rc.appliedIndex, &rc.confState, data)
	if err != nil {
		panic(err)
	}
	if err := rc.saveSnap(snap); err != nil {
		panic(err)
	}

	compactIndex := uint64(1)
	if rc.appliedIndex > snapshotCatchUpEntriesN {
		compactIndex = rc.appliedIndex - snapshotCatchUpEntriesN
	}
	if err := rc.raftStorage.Compact(compactIndex); err != nil {
		panic(err)
	}

	logtool.RLog.Info("compacted log at index ", map[string]interface{}{
		"index": compactIndex,
	})
	rc.snapshotIndex = rc.appliedIndex
}

func (rc *raftNode) serveChannels(electedCh chan bool, errCh chan error) {
	snap, err := rc.raftStorage.Snapshot()
	if err != nil {
		panic(err)
	}
	rc.confState = snap.Metadata.ConfState
	rc.snapshotIndex = snap.Metadata.Index
	rc.appliedIndex = snap.Metadata.Index

	defer rc.wal.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// send proposals over raft
	go func() {
		confChangeCount := uint64(0)

		for rc.proposeC != nil && rc.confChangeC != nil {
			select {
			case prop, ok := <-rc.proposeC:
				if !ok {
					rc.proposeC = nil
				} else {
					// blocks until accepted by raft state machine
					rc.node.Propose(context.TODO(), []byte(prop))
				}

			case cc, ok := <-rc.confChangeC:
				if !ok {
					rc.confChangeC = nil
				} else {
					confChangeCount++
					cc.ID = confChangeCount
					rc.node.ProposeConfChange(context.TODO(), cc)
				}
			}
		}
		// client closed channel; shutdown raft if not already
		close(rc.stopc)
	}()

	// event loop on raft state machine updates
	for {
		select {
		case <-ticker.C:
			rc.node.Tick()

		// store raft entries to wal, then publish over commit channel
		case rd := <-rc.node.Ready():
			if rd.SoftState != nil && rd.SoftState.Lead != raft.None {
				logtool.RLog.Info("node ready get message", map[string]interface{}{
					"leader id": rd.SoftState.Lead,
					"node id ":  rc.id,
					"node name": rc.nodeName,
				})
				if rc.id == rd.SoftState.Lead {
					electedCh <- true
				} else {
					electedCh <- false
				}
			} else if rd.SoftState != nil && rd.SoftState.Lead == raft.None {
				logtool.RLog.Error("leader down in exception", map[string]interface{}{
					"errno":  errhandle.E_LEADER_DOWN,
					"errmsg": errhandle.Msg[errhandle.E_LEADER_DOWN],
				})
			}
			rc.wal.Save(rd.HardState, rd.Entries)
			if !raft.IsEmptySnap(rd.Snapshot) {
				rc.saveSnap(rd.Snapshot)
				rc.raftStorage.ApplySnapshot(rd.Snapshot)
				rc.publishSnapshot(rd.Snapshot)
			}
			rc.raftStorage.Append(rd.Entries)
			rc.transport.Send(rd.Messages)
			if ok := rc.publishEntries(rc.entriesToApply(rd.CommittedEntries)); !ok {
				rc.stop()
				return
			}
			rc.maybeTriggerSnapshot()
			rc.node.Advance()

		case err := <-rc.transport.ErrorC:
			rc.writeError(err)
			errCh <- err
			return

		case <-rc.stopc:
			rc.stop()
			return
		}
	}
}

func (rc *raftNode) serveRaft() {
	url, err := url.Parse(rc.selfPeer)
	if err != nil {
		logtool.RLog.Fatal("raft: Failed parsing URL", map[string]interface{}{
			"error": err,
		})
	}

	ln, err := raftsvr.NewStoppableListener(url.Host, rc.httpstopc)
	if err != nil {
		logtool.RLog.Fatal("raft: Failed to listen rafthttp", map[string]interface{}{
			"error": err,
		})
	}

	err = (&http.Server{Handler: rc.transport.Handler()}).Serve(ln)
	select {
	case <-rc.httpstopc:
	default:
		logtool.RLog.Fatal("raft: Failed to serve rafthttp", map[string]interface{}{
			"error": err,
		})
	}
	close(rc.httpdonec)
}

func (rc *raftNode) Process(ctx context.Context, m raftpb.Message) error {
	return rc.node.Step(ctx, m)
}
func (rc *raftNode) IsIDRemoved(id uint64) bool                           { return false }
func (rc *raftNode) ReportUnreachable(id uint64)                          {}
func (rc *raftNode) ReportSnapshot(id uint64, status raft.SnapshotStatus) {}
