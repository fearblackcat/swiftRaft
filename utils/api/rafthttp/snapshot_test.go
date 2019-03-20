package rafthttp

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.xiaojukeji.com/gulfstream/dcron/workflow/logtool"
	"github.com/fearblackcat/smartRaft/raft/raftpb"
	"github.com/fearblackcat/smartRaft/utils/api/snap"
	"github.com/fearblackcat/smartRaft/utils/pkg/types"
)

type strReaderCloser struct{ *strings.Reader }

func (s strReaderCloser) Close() error { return nil }

func TestSnapshotSend(t *testing.T) {
	tests := []struct {
		m    raftpb.Message
		rc   io.ReadCloser
		size int64

		wsent  bool
		wfiles int
	}{
		// sent and receive with no errors
		{
			m:    raftpb.Message{Type: raftpb.MsgSnap, To: 1},
			rc:   strReaderCloser{strings.NewReader("hello")},
			size: 5,

			wsent:  true,
			wfiles: 1,
		},
		// error when reading snapshot for send
		{
			m:    raftpb.Message{Type: raftpb.MsgSnap, To: 1},
			rc:   &errReadCloser{fmt.Errorf("snapshot error")},
			size: 1,

			wsent:  false,
			wfiles: 0,
		},
		// sends less than the given snapshot length
		{
			m:    raftpb.Message{Type: raftpb.MsgSnap, To: 1},
			rc:   strReaderCloser{strings.NewReader("hello")},
			size: 10000,

			wsent:  false,
			wfiles: 0,
		},
		// sends less than actual snapshot length
		{
			m:    raftpb.Message{Type: raftpb.MsgSnap, To: 1},
			rc:   strReaderCloser{strings.NewReader("hello")},
			size: 1,

			wsent:  false,
			wfiles: 0,
		},
	}

	for i, tt := range tests {
		sent, files := testSnapshotSend(t, snap.NewMessage(tt.m, tt.rc, tt.size))
		if tt.wsent != sent {
			t.Errorf("#%d: snapshot expected %v, got %v", i, tt.wsent, sent)
		}
		if tt.wfiles != len(files) {
			t.Fatalf("#%d: expected %d files, got %d files", i, tt.wfiles, len(files))
		}
	}
}

func testSnapshotSend(t *testing.T, sm *snap.Message) (bool, []os.FileInfo) {
	d, err := ioutil.TempDir(os.TempDir(), "snapdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(d)

	r := &fakeRaft{}
	tr := &Transport{pipelineRt: &http.Transport{}, ClusterID: types.ID(1), Raft: r}
	ch := make(chan struct{}, 1)
	h := &syncHandler{newSnapshotHandler(tr, r, snap.New(logtool.RLog, d), types.ID(1)), ch}
	srv := httptest.NewServer(h)
	defer srv.Close()

	picker := mustNewURLPicker(t, []string{srv.URL})
	snapsend := newSnapshotSender(tr, picker, types.ID(1), newPeerStatus(logtool.RLog, types.ID(0), types.ID(1)))
	defer snapsend.stop()

	snapsend.send(*sm)

	sent := false
	select {
	case <-time.After(time.Second):
		t.Fatalf("timed out sending snapshot")
	case sent = <-sm.CloseNotify():
	}

	// wait for handler to finish accepting snapshot
	<-ch

	files, rerr := ioutil.ReadDir(d)
	if rerr != nil {
		t.Fatal(rerr)
	}
	return sent, files
}

type errReadCloser struct{ err error }

func (s *errReadCloser) Read(p []byte) (int, error) { return 0, s.err }
func (s *errReadCloser) Close() error               { return s.err }

type syncHandler struct {
	h  http.Handler
	ch chan<- struct{}
}

func (sh *syncHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sh.h.ServeHTTP(w, r)
	sh.ch <- struct{}{}
}
