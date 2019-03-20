package fileutil

import (
	"errors"
	"os"
)

var (
	ErrLocked = errors.New("fileutil: file already locked")
)

type LockedFile struct{ *os.File }
