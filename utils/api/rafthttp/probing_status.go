package rafthttp

import (
	"time"

	"git.xiaojukeji.com/gulfstream/dcron/workflow/logtool"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xiang90/probing"
)

const (
	// RoundTripperNameRaftMessage is the name of round-tripper that sends
	// all other Raft messages, other than "snap.Message".
	RoundTripperNameRaftMessage = "ROUND_TRIPPER_RAFT_MESSAGE"
	// RoundTripperNameSnapshot is the name of round-tripper that sends merged snapshot message.
	RoundTripperNameSnapshot = "ROUND_TRIPPER_SNAPSHOT"
)

var (
	// proberInterval must be shorter than read timeout.
	// Or the connection will time-out.
	proberInterval           = ConnReadTimeout - time.Second
	statusMonitoringInterval = 30 * time.Second
	statusErrorInterval      = 5 * time.Second
)

func addPeerToProber(lg *logtool.RLogHandle, p probing.Prober, id string, us []string, roundTripperName string, rttSecProm *prometheus.HistogramVec) {
	hus := make([]string, len(us))
	for i := range us {
		hus[i] = us[i] + ProbingPrefix
	}

	p.AddHTTP(id, proberInterval, hus)

	s, err := p.Status(id)
	if err != nil {
		if lg != nil {
			lg.Warn("failed to add peer into prober", map[string]interface{}{
				"remote-peer-id": id,
			})
		} else {
			plog.Errorf("failed to add peer %s into prober", id)
		}
		return
	}

	go monitorProbingStatus(lg, s, id, roundTripperName, rttSecProm)
}

func monitorProbingStatus(lg *logtool.RLogHandle, s probing.Status, id string, roundTripperName string, rttSecProm *prometheus.HistogramVec) {
	// set the first interval short to log error early.
	interval := statusErrorInterval
	for {
		select {
		case <-time.After(interval):
			if !s.Health() {
				if lg != nil {
					lg.Warn("prober detected unhealthy status", map[string]interface{}{
						"round-tripper-name": roundTripperName,
						"remote-peer-id": id,
						"rtt": s.SRTT(),
					})
				} else {
					plog.Warningf("health check for peer %s could not connect: %v", id, s.Err())
				}
				interval = statusErrorInterval
			} else {
				interval = statusMonitoringInterval
			}
			if s.ClockDiff() > time.Second {
				if lg != nil {
					lg.Warn("prober found high clock drift", map[string]interface{}{
						"round-tripper-name": roundTripperName,
						"remote-peer-id": id,
						"clock-drift": s.SRTT(),
						"rtt": s.ClockDiff(),
					})
				} else {
					plog.Warningf("the clock difference against peer %s is too high [%v > %v]", id, s.ClockDiff(), time.Second)
				}
			}
			rttSecProm.WithLabelValues(id).Observe(s.SRTT().Seconds())

		case <-s.StopNotify():
			return
		}
	}
}
