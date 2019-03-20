package ioutil

import (
	"bytes"
	"testing"
)

func TestLimitedBufferReaderRead(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 10))
	ln := 1
	lr := NewLimitedBufferReader(buf, ln)
	n, err := lr.Read(make([]byte, 10))
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if n != ln {
		t.Errorf("len(data read) = %d, want %d", n, ln)
	}
}
