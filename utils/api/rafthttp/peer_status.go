package rafthttp

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fearblackcat/swiftRaft/utils/logtool"
	"github.com/fearblackcat/swiftRaft/utils/pkg/types"
)

type failureType struct {
	source string
	action string
}

type peerStatus struct {
	lg     *logtool.RLogHandle
	local  types.ID
	id     types.ID
	mu     sync.Mutex // protect variables below
	active bool
	since  time.Time
}

func newPeerStatus(lg *logtool.RLogHandle, local, id types.ID) *peerStatus {
	return &peerStatus{lg: lg, local: local, id: id}
}

func (s *peerStatus) activate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		if s.lg != nil {
			s.lg.Info("peer became active", map[string]interface{}{
				"peer-id": s.id.String(),
			})
		} else {
			plog.Infof("peer %s became active", s.id)
		}
		s.active = true
		s.since = time.Now()

		activePeers.WithLabelValues(s.local.String(), s.id.String()).Inc()
	}
}

func (s *peerStatus) deactivate(failure failureType, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := fmt.Sprintf("failed to %s %s on %s (%s)", failure.action, s.id, failure.source, reason)
	if s.active {
		if s.lg != nil {
			s.lg.Warn("peer became inactive (message send to peer failed)", map[string]interface{}{
				"peer-id": s.id.String(),
				"error":   errors.New(msg),
			})
		} else {
			plog.Errorf(msg)
			plog.Infof("peer %s became inactive (message send to peer failed)", s.id)
		}
		s.active = false
		s.since = time.Time{}

		activePeers.WithLabelValues(s.local.String(), s.id.String()).Dec()
		disconnectedPeers.WithLabelValues(s.local.String(), s.id.String()).Inc()
		return
	}

	if s.lg != nil {
		s.lg.Debug("peer deactivated again", map[string]interface{}{
			"peer-id": s.id.String(),
			"error":   errors.New(msg),
		})
	}
}

func (s *peerStatus) isActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

func (s *peerStatus) activeSince() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.since
}
