package fileutil

import (
	"io"
	"os"
)

// Preallocate tries to allocate the space for given
// file. This operation is only supported on linux by a
// few filesystems (btrfs, ext4, etc.).
// If the operation is unsupported, no error will be returned.
// Otherwise, the error encountered will be returned.
func Preallocate(f *os.File, sizeInBytes int64, extendFile bool) error {
	if sizeInBytes == 0 {
		// fallocate will return EINVAL if length is 0; skip
		return nil
	}
	if extendFile {
		return preallocExtend(f, sizeInBytes)
	}
	return preallocFixed(f, sizeInBytes)
}

func preallocExtendTrunc(f *os.File, sizeInBytes int64) error {
	curOff, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	size, err := f.Seek(sizeInBytes, io.SeekEnd)
	if err != nil {
		return err
	}
	if _, err = f.Seek(curOff, io.SeekStart); err != nil {
		return err
	}
	if sizeInBytes > size {
		return nil
	}
	return f.Truncate(sizeInBytes)
}
