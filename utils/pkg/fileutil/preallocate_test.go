package fileutil

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestPreallocateExtend(t *testing.T) {
	pf := func(f *os.File, sz int64) error { return Preallocate(f, sz, true) }
	tf := func(t *testing.T, f *os.File) { testPreallocateExtend(t, f, pf) }
	runPreallocTest(t, tf)
}

func TestPreallocateExtendTrunc(t *testing.T) {
	tf := func(t *testing.T, f *os.File) { testPreallocateExtend(t, f, preallocExtendTrunc) }
	runPreallocTest(t, tf)
}

func testPreallocateExtend(t *testing.T, f *os.File, pf func(*os.File, int64) error) {
	size := int64(64 * 1000)
	if err := pf(f, size); err != nil {
		t.Fatal(err)
	}

	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() != size {
		t.Errorf("size = %d, want %d", stat.Size(), size)
	}
}

func TestPreallocateFixed(t *testing.T) { runPreallocTest(t, testPreallocateFixed) }
func testPreallocateFixed(t *testing.T, f *os.File) {
	size := int64(64 * 1000)
	if err := Preallocate(f, size, false); err != nil {
		t.Fatal(err)
	}

	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() != 0 {
		t.Errorf("size = %d, want %d", stat.Size(), 0)
	}
}

func runPreallocTest(t *testing.T, test func(*testing.T, *os.File)) {
	p, err := ioutil.TempDir(os.TempDir(), "preallocateTest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(p)

	f, err := ioutil.TempFile(p, "")
	if err != nil {
		t.Fatal(err)
	}
	test(t, f)
}
