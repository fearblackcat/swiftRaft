package ioutil

import (
	"bytes"
	"io"
	"testing"
)

type readerNilCloser struct{ io.Reader }

func (rc *readerNilCloser) Close() error { return nil }

// TestExactReadCloserExpectEOF expects an eof when reading too much.
func TestExactReadCloserExpectEOF(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 10))
	rc := NewExactReadCloser(&readerNilCloser{buf}, 1)
	if _, err := rc.Read(make([]byte, 10)); err != ErrExpectEOF {
		t.Fatalf("expected %v, got %v", ErrExpectEOF, err)
	}
}

// TestExactReadCloserShort expects an eof when reading too little
func TestExactReadCloserShort(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 5))
	rc := NewExactReadCloser(&readerNilCloser{buf}, 10)
	if _, err := rc.Read(make([]byte, 10)); err != nil {
		t.Fatalf("Read expected nil err, got %v", err)
	}
	if err := rc.Close(); err != ErrShortRead {
		t.Fatalf("Close expected %v, got %v", ErrShortRead, err)
	}
}
