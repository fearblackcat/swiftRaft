package fileutil

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fearblackcat/swiftRaft/utils/logtool"
)

func PurgeFile(lg *logtool.RLogHandle, dirname string, suffix string, max uint, interval time.Duration, stop <-chan struct{}) <-chan error {
	return purgeFile(lg, dirname, suffix, max, interval, stop, nil)
}

// purgeFile is the internal implementation for PurgeFile which can post purged files to purgec if non-nil.
func purgeFile(lg *logtool.RLogHandle, dirname string, suffix string, max uint, interval time.Duration, stop <-chan struct{}, purgec chan<- string) <-chan error {
	errC := make(chan error, 1)
	go func() {
		for {
			fnames, err := ReadDir(dirname)
			if err != nil {
				errC <- err
				return
			}
			newfnames := make([]string, 0)
			for _, fname := range fnames {
				if strings.HasSuffix(fname, suffix) {
					newfnames = append(newfnames, fname)
				}
			}
			sort.Strings(newfnames)
			fnames = newfnames
			for len(newfnames) > int(max) {
				f := filepath.Join(dirname, newfnames[0])
				l, err := TryLockFile(f, os.O_WRONLY, PrivateFileMode)
				if err != nil {
					break
				}
				if err = os.Remove(f); err != nil {
					errC <- err
					return
				}
				if err = l.Close(); err != nil {
					if lg != nil {
						lg.Warn("failed to unlock/close", map[string]interface{}{"path": l.Name(), "error": err.Error()})
					} else {
						plog.Errorf("error unlocking %s when purging file (%v)", l.Name(), err)
					}
					errC <- err
					return
				}
				if lg != nil {
					lg.Info("purged", map[string]interface{}{"path": f})
				} else {
					plog.Infof("purged file %s successfully", f)
				}
				newfnames = newfnames[1:]
			}
			if purgec != nil {
				for i := 0; i < len(fnames)-len(newfnames); i++ {
					purgec <- fnames[i]
				}
			}
			select {
			case <-time.After(interval):
			case <-stop:
				return
			}
		}
	}()
	return errC
}
