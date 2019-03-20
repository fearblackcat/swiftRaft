package ioutil

import (
	"fmt"
	"io"
)

// ReaderAndCloser implements io.ReadCloser interface by combining
// reader and closer together.
type ReaderAndCloser struct {
	io.Reader
	io.Closer
}

var (
	ErrShortRead = fmt.Errorf("ioutil: short read")
	ErrExpectEOF = fmt.Errorf("ioutil: expect EOF")
)

// NewExactReadCloser returns a ReadCloser that returns errors if the underlying
// reader does not read back exactly the requested number of bytes.
func NewExactReadCloser(rc io.ReadCloser, totalBytes int64) io.ReadCloser {
	return &exactReadCloser{rc: rc, totalBytes: totalBytes}
}

type exactReadCloser struct {
	rc         io.ReadCloser
	br         int64
	totalBytes int64
}

func (e *exactReadCloser) Read(p []byte) (int, error) {
	n, err := e.rc.Read(p)
	e.br += int64(n)
	if e.br > e.totalBytes {
		return 0, ErrExpectEOF
	}
	if e.br < e.totalBytes && n == 0 {
		return 0, ErrShortRead
	}
	return n, err
}

func (e *exactReadCloser) Close() error {
	if err := e.rc.Close(); err != nil {
		return err
	}
	if e.br < e.totalBytes {
		return ErrShortRead
	}
	return nil
}
