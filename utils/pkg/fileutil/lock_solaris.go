// +build solaris

package fileutil

import (
	"os"
	"syscall"
)

func TryLockFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	var lock syscall.Flock_t
	lock.Start = 0
	lock.Len = 0
	lock.Pid = 0
	lock.Type = syscall.F_WRLCK
	lock.Whence = 0
	lock.Pid = 0
	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, err
	}
	if err := syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, &lock); err != nil {
		f.Close()
		if err == syscall.EAGAIN {
			err = ErrLocked
		}
		return nil, err
	}
	return &LockedFile{f}, nil
}

func LockFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	var lock syscall.Flock_t
	lock.Start = 0
	lock.Len = 0
	lock.Pid = 0
	lock.Type = syscall.F_WRLCK
	lock.Whence = 0
	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, err
	}
	if err = syscall.FcntlFlock(f.Fd(), syscall.F_SETLKW, &lock); err != nil {
		f.Close()
		return nil, err
	}
	return &LockedFile{f}, nil
}
