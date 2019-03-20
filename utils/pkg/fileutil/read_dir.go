package fileutil

import (
	"os"
	"path/filepath"
	"sort"
)

// ReadDirOp represents an read-directory operation.
type ReadDirOp struct {
	ext string
}

// ReadDirOption configures archiver operations.
type ReadDirOption func(*ReadDirOp)

// WithExt filters file names by their extensions.
// (e.g. WithExt(".wal") to list only WAL files)
func WithExt(ext string) ReadDirOption {
	return func(op *ReadDirOp) { op.ext = ext }
}

func (op *ReadDirOp) applyOpts(opts []ReadDirOption) {
	for _, opt := range opts {
		opt(op)
	}
}

// ReadDir returns the filenames in the given directory in sorted order.
func ReadDir(d string, opts ...ReadDirOption) ([]string, error) {
	op := &ReadDirOp{}
	op.applyOpts(opts)

	dir, err := os.Open(d)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)

	if op.ext != "" {
		tss := make([]string, 0)
		for _, v := range names {
			if filepath.Ext(v) == op.ext {
				tss = append(tss, v)
			}
		}
		names = tss
	}
	return names, nil
}
