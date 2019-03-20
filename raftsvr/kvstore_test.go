package raftsvr

import (
	"reflect"
	"testing"
)

func Test_kvstore_snapshot(t *testing.T) {
	tm := map[string]string{"foo": "bar"}
	s := &Kvstore{KvStore: tm}

	v, _ := s.Lookup("foo")
	if v != "bar" {
		t.Fatalf("foo has unexpected value, got %s", v)
	}

	data, err := s.GetSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	s.KvStore = nil

	if err := s.RecoverFromSnapshot(data); err != nil {
		t.Fatal(err)
	}
	v, _ = s.Lookup("foo")
	if v != "bar" {
		t.Fatalf("foo has unexpected value, got %s", v)
	}
	if !reflect.DeepEqual(s.KvStore, tm) {
		t.Fatalf("store expected %+v, got %+v", tm, s.KvStore)
	}
}
