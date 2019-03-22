package rafthttp

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/fearblackcat/swiftRaft/raft"
	"github.com/fearblackcat/swiftRaft/utils/api/snap"
	"github.com/fearblackcat/swiftRaft/utils/logtool"
	"github.com/fearblackcat/swiftRaft/utils/pkg/httputil"
	pioutil "github.com/fearblackcat/swiftRaft/utils/pkg/ioutil"
	"github.com/fearblackcat/swiftRaft/utils/pkg/types"

	"github.com/dustin/go-humanize"
)

var (
	// timeout for reading snapshot response body
	snapResponseReadTimeout = 5 * time.Second
)

type snapshotSender struct {
	from, to types.ID
	cid      types.ID

	tr     *Transport
	picker *urlPicker
	status *peerStatus
	r      Raft
	errorc chan error

	stopc chan struct{}
}

func newSnapshotSender(tr *Transport, picker *urlPicker, to types.ID, status *peerStatus) *snapshotSender {
	return &snapshotSender{
		from:   tr.ID,
		to:     to,
		cid:    tr.ClusterID,
		tr:     tr,
		picker: picker,
		status: status,
		r:      tr.Raft,
		errorc: tr.ErrorC,
		stopc:  make(chan struct{}),
	}
}

func (s *snapshotSender) stop() { close(s.stopc) }

func (s *snapshotSender) send(merged snap.Message) {
	start := time.Now()

	m := merged.Message
	to := types.ID(m.To).String()

	body := createSnapBody(s.tr.Logger, merged)
	defer body.Close()

	u := s.picker.pick()
	req := createPostRequest(u, RaftSnapshotPrefix, body, "application/octet-stream", s.tr.URLs, s.from, s.cid)

	if s.tr.Logger != nil {
		s.tr.Logger.Info("sending database snapshot", map[string]interface{}{
			"snapshot-index": m.Snapshot.Metadata.Index,
			"remote-peer-id": to,
			"bytes":          merged.TotalSize,
			"size":           humanize.Bytes(uint64(merged.TotalSize)),
		})
	} else {
		plog.Infof("start to send database snapshot [index: %d, to %s]...", m.Snapshot.Metadata.Index, types.ID(m.To))
	}

	err := s.post(req)
	defer merged.CloseWithError(err)
	if err != nil {
		if s.tr.Logger != nil {
			s.tr.Logger.Warn("failed to send database snapshot", map[string]interface{}{
				"snapshot-index": m.Snapshot.Metadata.Index,
				"remote-peer-id": to,
				"bytes":          merged.TotalSize,
				"size":           humanize.Bytes(uint64(merged.TotalSize)),
			})
		} else {
			plog.Warningf("database snapshot [index: %d, to: %s] failed to be sent out (%v)", m.Snapshot.Metadata.Index, types.ID(m.To), err)
		}

		// errMemberRemoved is a critical error since a removed member should
		// always be stopped. So we use reportCriticalError to report it to errorc.
		if err == errMemberRemoved {
			reportCriticalError(err, s.errorc)
		}

		s.picker.unreachable(u)
		s.status.deactivate(failureType{source: sendSnap, action: "post"}, err.Error())
		s.r.ReportUnreachable(m.To)
		// report SnapshotFailure to raft state machine. After raft state
		// machine knows about it, it would pause a while and retry sending
		// new snapshot message.
		s.r.ReportSnapshot(m.To, raft.SnapshotFailure)
		sentFailures.WithLabelValues(to).Inc()
		snapshotSendFailures.WithLabelValues(to).Inc()
		return
	}
	s.status.activate()
	s.r.ReportSnapshot(m.To, raft.SnapshotFinish)

	if s.tr.Logger != nil {
		s.tr.Logger.Info("sent database snapshot", map[string]interface{}{
			"snapshot-index": m.Snapshot.Metadata.Index,
			"remote-peer-id": to,
			"bytes":          merged.TotalSize,
			"size":           humanize.Bytes(uint64(merged.TotalSize)),
		})
	} else {
		plog.Infof("database snapshot [index: %d, to: %s] sent out successfully", m.Snapshot.Metadata.Index, types.ID(m.To))
	}

	sentBytes.WithLabelValues(to).Add(float64(merged.TotalSize))

	snapshotSend.WithLabelValues(to).Inc()
	snapshotSendSeconds.WithLabelValues(to).Observe(time.Since(start).Seconds())
}

// post posts the given request.
// It returns nil when request is sent out and processed successfully.
func (s *snapshotSender) post(req *http.Request) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	defer cancel()

	type responseAndError struct {
		resp *http.Response
		body []byte
		err  error
	}
	result := make(chan responseAndError, 1)

	go func() {
		resp, err := s.tr.pipelineRt.RoundTrip(req)
		if err != nil {
			result <- responseAndError{resp, nil, err}
			return
		}

		// close the response body when timeouts.
		// prevents from reading the body forever when the other side dies right after
		// successfully receives the request body.
		time.AfterFunc(snapResponseReadTimeout, func() { httputil.GracefulClose(resp) })
		body, err := ioutil.ReadAll(resp.Body)
		result <- responseAndError{resp, body, err}
	}()

	select {
	case <-s.stopc:
		return errStopped
	case r := <-result:
		if r.err != nil {
			return r.err
		}
		return checkPostResponse(r.resp, r.body, req, s.to)
	}
}

func createSnapBody(lg *logtool.RLogHandle, merged snap.Message) io.ReadCloser {
	buf := new(bytes.Buffer)
	enc := &messageEncoder{w: buf}
	// encode raft message
	if err := enc.encode(&merged.Message); err != nil {
		if lg != nil {
			lg.Panic("failed to encode message", map[string]interface{}{
				"error": err,
			})
		} else {
			plog.Panicf("encode message error (%v)", err)
		}
	}

	return &pioutil.ReaderAndCloser{
		Reader: io.MultiReader(buf, merged.ReadCloser),
		Closer: merged.ReadCloser,
	}
}
