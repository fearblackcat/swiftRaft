package raftsvr

import (
	"sync"
	"time"

	"github.com/fearblackcat/swiftRaft/utils/logtool"
	"github.com/fearblackcat/swiftRaft/utils/pkg/fileutil"
)

const (
	DefaultMaxSnapshots = 5
	DefaultMaxWALs      = 5

	purgeFileInterval = 30 * time.Second
)

type ServerAttach struct {
	MaxSnapFiles uint
	MaxWALFiles  uint
	SnapDir      string
	WalDir       string
	Stopping     chan struct{}
	Done         chan struct{}
	WgMu         sync.RWMutex
	Wg           sync.WaitGroup
}

func NewServerAttach(waldir, snapdir string, stop, done chan struct{}) *ServerAttach {
	srv := &ServerAttach{}

	srv.Stopping = stop
	srv.Done = done
	srv.SnapDir = snapdir
	srv.WalDir = waldir
	srv.MaxSnapFiles = DefaultMaxSnapshots
	srv.MaxWALFiles = DefaultMaxWALs

	return srv
}

func (s *ServerAttach) GoAttach(f func()) {
	s.WgMu.RLock() // this blocks with ongoing close(s.stopping)
	defer s.WgMu.RUnlock()
	select {
	case <-s.Stopping:
		logtool.NLog.Warn("server has stopped; skipping goAttach")
		return
	default:
	}

	// now safe to add since waitgroup wait has not started yet
	s.Wg.Add(1)
	go func() {
		defer s.Wg.Done()
		f()
	}()
	s.Wg.Wait()
}

func (s *ServerAttach) PurgeFile() {
	var serrc, werrc <-chan error
	if s.MaxSnapFiles > 0 {
		//dberrc = fileutil.PurgeFile(logtool.RLog, s.SnapDir, "snap.db", s.MaxSnapFiles, purgeFileInterval, s.Done)
		serrc = fileutil.PurgeFile(logtool.RLog, s.SnapDir, "snap", s.MaxSnapFiles, purgeFileInterval, s.Done)
	}
	if s.MaxWALFiles > 0 {
		werrc = fileutil.PurgeFile(logtool.RLog, s.WalDir, "wal", s.MaxWALFiles, purgeFileInterval, s.Done)
	}

	select {
	case e := <-serrc:
		logtool.RLog.Fatal("failed to purge snap file", map[string]interface{}{"error": e.Error()})
	case e := <-werrc:
		logtool.RLog.Fatal("failed to purge wal file", map[string]interface{}{"error": e.Error()})
	case <-s.Stopping:
		return
	}
}
