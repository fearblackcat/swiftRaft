package fileutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadDir(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	defer os.RemoveAll(tmpdir)
	if err != nil {
		t.Fatalf("unexpected ioutil.TempDir error: %v", err)
	}

	files := []string{"def", "abc", "xyz", "ghi"}
	for _, f := range files {
		writeFunc(t, filepath.Join(tmpdir, f))
	}
	fs, err := ReadDir(tmpdir)
	if err != nil {
		t.Fatalf("error calling ReadDir: %v", err)
	}
	wfs := []string{"abc", "def", "ghi", "xyz"}
	if !reflect.DeepEqual(fs, wfs) {
		t.Fatalf("ReadDir: got %v, want %v", fs, wfs)
	}

	files = []string{"def.wal", "abc.wal", "xyz.wal", "ghi.wal"}
	for _, f := range files {
		writeFunc(t, filepath.Join(tmpdir, f))
	}
	fs, err = ReadDir(tmpdir, WithExt(".wal"))
	if err != nil {
		t.Fatalf("error calling ReadDir: %v", err)
	}
	wfs = []string{"abc.wal", "def.wal", "ghi.wal", "xyz.wal"}
	if !reflect.DeepEqual(fs, wfs) {
		t.Fatalf("ReadDir: got %v, want %v", fs, wfs)
	}
}

func writeFunc(t *testing.T, path string) {
	fh, err := os.Create(path)
	if err != nil {
		t.Fatalf("error creating file: %v", err)
	}
	if err = fh.Close(); err != nil {
		t.Fatalf("error closing file: %v", err)
	}
}
