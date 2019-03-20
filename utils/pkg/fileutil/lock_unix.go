// +build !windows,!plan9,!solaris,!linux

package fileutil

import (
	"os"
)

func TryLockFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	return flockTryLockFile(path, flag, perm)
}

func LockFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	return flockLockFile(path, flag, perm)
}
