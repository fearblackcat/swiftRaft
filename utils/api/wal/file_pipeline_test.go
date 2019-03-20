package wal

import (
	"io/ioutil"
	"math"
	"os"
	"testing"

	"git.xiaojukeji.com/gulfstream/dcron/workflow/logtool"
)

func TestFilePipeline(t *testing.T) {
	tdir, err := ioutil.TempDir(os.TempDir(), "wal-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tdir)

	fp := newFilePipeline(logtool.RLog, tdir, SegmentSizeBytes)
	defer fp.Close()

	f, ferr := fp.Open()
	if ferr != nil {
		t.Fatal(ferr)
	}
	f.Close()
}

func TestFilePipelineFailPreallocate(t *testing.T) {
	tdir, err := ioutil.TempDir(os.TempDir(), "wal-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tdir)

	fp := newFilePipeline(logtool.RLog, tdir, math.MaxInt64)
	defer fp.Close()

	f, ferr := fp.Open()
	if f != nil || ferr == nil { // no space left on device
		t.Fatal("expected error on invalid pre-allocate size, but no error")
	}
}

func TestFilePipelineFailLockFile(t *testing.T) {
	tdir, err := ioutil.TempDir(os.TempDir(), "wal-test")
	if err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(tdir)

	fp := newFilePipeline(logtool.RLog, tdir, math.MaxInt64)
	defer fp.Close()

	f, ferr := fp.Open()
	if f != nil || ferr == nil { // no such file or directory
		t.Fatal("expected error on invalid pre-allocate size, but no error")
	}
}
