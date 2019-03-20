// +build !linux,!darwin

package fileutil

import "os"

func preallocExtend(f *os.File, sizeInBytes int64) error {
	return preallocExtendTrunc(f, sizeInBytes)
}

func preallocFixed(f *os.File, sizeInBytes int64) error { return nil }
