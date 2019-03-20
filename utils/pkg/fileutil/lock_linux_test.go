// +build linux

package fileutil

import "testing"

// TestLockAndUnlockSyscallFlock tests the fallback flock using the flock syscall.
func TestLockAndUnlockSyscallFlock(t *testing.T) {
	oldTryLock, oldLock := linuxTryLockFile, linuxLockFile
	defer func() {
		linuxTryLockFile, linuxLockFile = oldTryLock, oldLock
	}()
	linuxTryLockFile, linuxLockFile = flockTryLockFile, flockLockFile
	TestLockAndUnlock(t)
}
