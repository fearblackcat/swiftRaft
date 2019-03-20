package rafthttp

import (
	"git.xiaojukeji.com/gulfstream/dcron/workflow/logtool"
	"github.com/fearblackcat/smartRaft/raft/raftpb"
	"github.com/fearblackcat/smartRaft/utils/pkg/types"
)

type remote struct {
	lg       *logtool.RLogHandle
	localID  types.ID
	id       types.ID
	status   *peerStatus
	pipeline *pipeline
}

func startRemote(tr *Transport, urls types.URLs, id types.ID) *remote {
	picker := newURLPicker(urls)
	status := newPeerStatus(tr.Logger, tr.ID, id)
	pipeline := &pipeline{
		peerID: id,
		tr:     tr,
		picker: picker,
		status: status,
		raft:   tr.Raft,
		errorc: tr.ErrorC,
	}
	pipeline.start()

	return &remote{
		lg:       tr.Logger,
		localID:  tr.ID,
		id:       id,
		status:   status,
		pipeline: pipeline,
	}
}

func (g *remote) send(m raftpb.Message) {
	select {
	case g.pipeline.msgc <- m:
	default:
		if g.status.isActive() {
			if g.lg != nil {
				g.lg.Warn("dropped internal Raft message since sending buffer is full (overloaded network)",
					map[string]interface{}{
						"message-type":       m.Type.String(),
						"local-member-id":    g.localID.String(),
						"from":               types.ID(m.From).String(),
						"remote-peer-id":     g.id.String(),
						"remote-peer-active": g.status.isActive(),
					})
			} else {
				plog.MergeWarningf("dropped internal raft message to %s since sending buffer is full (bad/overloaded network)", g.id)
			}
		} else {
			if g.lg != nil {
				g.lg.Warn("dropped Raft message since sending buffer is full (overloaded network)",
					map[string]interface{}{
						"message-type":       m.Type.String(),
						"local-member-id":    g.localID.String(),
						"from":               types.ID(m.From).String(),
						"remote-peer-id":     g.id.String(),
						"remote-peer-active": g.status.isActive(),
					})
			} else {
				plog.Debugf("dropped %s to %s since sending buffer is full", m.Type, g.id)
			}
		}
		sentFailures.WithLabelValues(types.ID(m.To).String()).Inc()
	}
}

func (g *remote) stop() {
	g.pipeline.stop()
}

func (g *remote) Pause() {
	g.stop()
}

func (g *remote) Resume() {
	g.pipeline.start()
}
