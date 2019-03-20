package wal

import (
	"fmt"
	"os"
	"path/filepath"

	"git.xiaojukeji.com/gulfstream/dcron/workflow/logtool"
	"github.com/fearblackcat/smartRaft/utils/pkg/fileutil"
)

// filePipeline pipelines allocating disk space
type filePipeline struct {
	lg *logtool.RLogHandle

	// dir to put files
	dir string
	// size of files to make, in bytes
	size int64
	// count number of files generated
	count int

	filec chan *fileutil.LockedFile
	errc  chan error
	donec chan struct{}
}

func newFilePipeline(lg *logtool.RLogHandle, dir string, fileSize int64) *filePipeline {
	fp := &filePipeline{
		lg:    lg,
		dir:   dir,
		size:  fileSize,
		filec: make(chan *fileutil.LockedFile),
		errc:  make(chan error, 1),
		donec: make(chan struct{}),
	}
	go fp.run()
	return fp
}

// Open returns a fresh file for writing. Rename the file before calling
// Open again or there will be file collisions.
func (fp *filePipeline) Open() (f *fileutil.LockedFile, err error) {
	select {
	case f = <-fp.filec:
	case err = <-fp.errc:
	}
	return f, err
}

func (fp *filePipeline) Close() error {
	close(fp.donec)
	return <-fp.errc
}

func (fp *filePipeline) alloc() (f *fileutil.LockedFile, err error) {
	// count % 2 so this file isn't the same as the one last published
	fpath := filepath.Join(fp.dir, fmt.Sprintf("%d.tmp", fp.count%2))
	if f, err = fileutil.LockFile(fpath, os.O_CREATE|os.O_WRONLY, fileutil.PrivateFileMode); err != nil {
		return nil, err
	}
	if err = fileutil.Preallocate(f.File, fp.size, true); err != nil {
		if fp.lg != nil {
			fp.lg.Warn("failed to preallocate space when creating a new WAL", map[string]interface{}{
				"size":  fp.size,
				"error": err,
			})
		} else {
			plog.Errorf("failed to allocate space when creating new wal file (%v)", err)
		}
		f.Close()
		return nil, err
	}
	fp.count++
	return f, nil
}

func (fp *filePipeline) run() {
	defer close(fp.errc)
	for {
		f, err := fp.alloc()
		if err != nil {
			fp.errc <- err
			return
		}
		select {
		case fp.filec <- f:
		case <-fp.donec:
			os.Remove(f.Name())
			f.Close()
			return
		}
	}
}
