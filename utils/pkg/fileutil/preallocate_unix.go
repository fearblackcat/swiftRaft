// +build linux

package fileutil

import (
	"os"
	"syscall"
)

func preallocExtend(f *os.File, sizeInBytes int64) error {
	// use mode = 0 to change size
	err := syscall.Fallocate(int(f.Fd()), 0, 0, sizeInBytes)
	if err != nil {
		errno, ok := err.(syscall.Errno)
		// not supported; fallback
		// fallocate EINTRs frequently in some environments; fallback
		if ok && (errno == syscall.ENOTSUP || errno == syscall.EINTR) {
			return preallocExtendTrunc(f, sizeInBytes)
		}
	}
	return err
}

func preallocFixed(f *os.File, sizeInBytes int64) error {
	// use mode = 1 to keep size; see FALLOC_FL_KEEP_SIZE
	err := syscall.Fallocate(int(f.Fd()), 1, 0, sizeInBytes)
	if err != nil {
		errno, ok := err.(syscall.Errno)
		// treat not supported as nil error
		if ok && errno == syscall.ENOTSUP {
			return nil
		}
	}
	return err
}
