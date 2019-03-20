package fileutil

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestLockAndUnlock(t *testing.T) {
	f, err := ioutil.TempFile("", "lock")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer func() {
		err = os.Remove(f.Name())
		if err != nil {
			t.Fatal(err)
		}
	}()

	// lock the file
	l, err := LockFile(f.Name(), os.O_WRONLY, PrivateFileMode)
	if err != nil {
		t.Fatal(err)
	}

	// try lock a locked file
	if _, err = TryLockFile(f.Name(), os.O_WRONLY, PrivateFileMode); err != ErrLocked {
		t.Fatal(err)
	}

	// unlock the file
	if err = l.Close(); err != nil {
		t.Fatal(err)
	}

	// try lock the unlocked file
	dupl, err := TryLockFile(f.Name(), os.O_WRONLY, PrivateFileMode)
	if err != nil {
		t.Errorf("err = %v, want %v", err, nil)
	}

	// blocking on locked file
	locked := make(chan struct{}, 1)
	go func() {
		bl, blerr := LockFile(f.Name(), os.O_WRONLY, PrivateFileMode)
		if blerr != nil {
			t.Fatal(blerr)
		}
		locked <- struct{}{}
		if blerr = bl.Close(); blerr != nil {
			t.Fatal(blerr)
		}
	}()

	select {
	case <-locked:
		t.Error("unexpected unblocking")
	case <-time.After(100 * time.Millisecond):
	}

	// unlock
	if err = dupl.Close(); err != nil {
		t.Fatal(err)
	}

	// the previously blocked routine should be unblocked
	select {
	case <-locked:
	case <-time.After(1 * time.Second):
		t.Error("unexpected blocking")
	}
}
