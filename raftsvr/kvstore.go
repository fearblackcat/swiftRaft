package raftsvr

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"
	"sync"

	"github.com/fearblackcat/smartRaft/utils/api/snap"
)

// a key-value store backed by raft
type Kvstore struct {
	ProposeC    chan<- string // channel for proposing updates
	Mu          sync.RWMutex
	KvStore     map[string]string // current committed key-value pairs
	Snapshotter *snap.Snapshotter
}

type Kv struct {
	Key string
	Val string
}

func NewKVStore(snapshotter *snap.Snapshotter, proposeC chan<- string) *Kvstore {
	s := &Kvstore{ProposeC: proposeC, KvStore: make(map[string]string), Snapshotter: snapshotter}

	return s
}

func (s *Kvstore) LoadDataToMap(commitC <-chan *string, errorC <-chan error) {
	// replay log into key-value map
	s.ReadCommits(commitC, errorC)
	// read commits from raft into kvStore map until error
	go s.ReadCommits(commitC, errorC)
}

func (s *Kvstore) Lookup(key string) (string, bool) {
	s.Mu.RLock()
	v, ok := s.KvStore[key]
	s.Mu.RUnlock()
	return v, ok
}

func (s *Kvstore) Propose(k string, v string) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(Kv{k, v}); err != nil {
		log.Fatal(err)
	}
	s.ProposeC <- buf.String()
}

func (s *Kvstore) ReadCommits(commitC <-chan *string, errorC <-chan error) {
	for data := range commitC {
		if data == nil {
			// done replaying log; new data incoming
			// OR signaled to load snapshot
			snapshot, err := s.Snapshotter.Load()
			if err == snap.ErrNoSnapshot {
				return
			}
			if err != nil {
				log.Panic(err)
			}
			log.Printf("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
			if err := s.RecoverFromSnapshot(snapshot.Data); err != nil {
				log.Panic(err)
			}
			continue
		}

		var dataKv Kv
		dec := gob.NewDecoder(bytes.NewBufferString(*data))
		if err := dec.Decode(&dataKv); err != nil {
			log.Fatalf("dcron raft: could not decode message (%v)", err)
		}
		s.Mu.Lock()
		s.KvStore[dataKv.Key] = dataKv.Val
		s.Mu.Unlock()
	}
	if err, ok := <-errorC; ok {
		log.Fatal(err)
	}
}

func (s *Kvstore) GetSnapshot() ([]byte, error) {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	return json.Marshal(s.KvStore)
}

func (s *Kvstore) RecoverFromSnapshot(snapshot []byte) error {
	var store map[string]string
	if err := json.Unmarshal(snapshot, &store); err != nil {
		return err
	}
	s.Mu.Lock()
	s.KvStore = store
	s.Mu.Unlock()
	return nil
}
