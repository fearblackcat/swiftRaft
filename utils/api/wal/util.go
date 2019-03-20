package wal

import (
	"errors"
	"fmt"
	"strings"

	"git.xiaojukeji.com/gulfstream/dcron/workflow/logtool"
	"github.com/fearblackcat/smartRaft/utils/pkg/fileutil"
)

var errBadWALName = errors.New("bad wal name")

// Exist returns true if there are any files in a given directory.
func Exist(dir string) bool {
	names, err := fileutil.ReadDir(dir, fileutil.WithExt(".wal"))
	if err != nil {
		return false
	}
	return len(names) != 0
}

// searchIndex returns the last array index of names whose raft index section is
// equal to or smaller than the given index.
// The given names MUST be sorted.
func searchIndex(lg *logtool.RLogHandle, names []string, index uint64) (int, bool) {
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		_, curIndex, err := parseWALName(name)
		if err != nil {
			if lg != nil {
				lg.Panic("failed to parse WAL file name", map[string]interface{}{
					"path":  name,
					"error": err,
				})
			} else {
				plog.Panicf("parse correct name should never fail: %v", err)
			}
		}
		if index >= curIndex {
			return i, true
		}
	}
	return -1, false
}

// names should have been sorted based on sequence number.
// isValidSeq checks whether seq increases continuously.
func isValidSeq(lg *logtool.RLogHandle, names []string) bool {
	var lastSeq uint64
	for _, name := range names {
		curSeq, _, err := parseWALName(name)
		if err != nil {
			if lg != nil {
				lg.Panic("failed to parse WAL file name", map[string]interface{}{
					"path":  name,
					"error": err,
				})
			} else {
				plog.Panicf("parse correct name should never fail: %v", err)
			}
		}
		if lastSeq != 0 && lastSeq != curSeq-1 {
			return false
		}
		lastSeq = curSeq
	}
	return true
}

func readWALNames(lg *logtool.RLogHandle, dirpath string) ([]string, error) {
	names, err := fileutil.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}
	wnames := checkWalNames(lg, names)
	if len(wnames) == 0 {
		return nil, ErrFileNotFound
	}
	return wnames, nil
}

func checkWalNames(lg *logtool.RLogHandle, names []string) []string {
	wnames := make([]string, 0)
	for _, name := range names {
		if _, _, err := parseWALName(name); err != nil {
			// don't complain about left over tmp files
			if !strings.HasSuffix(name, ".tmp") {
				if lg != nil {
					lg.Warn("ignored file in WAL directory", map[string]interface{}{
						"path": name,
					})
				} else {
					plog.Warningf("ignored file %v in wal", name)
				}
			}
			continue
		}
		wnames = append(wnames, name)
	}
	return wnames
}

func parseWALName(str string) (seq, index uint64, err error) {
	if !strings.HasSuffix(str, ".wal") {
		return 0, 0, errBadWALName
	}
	_, err = fmt.Sscanf(str, "%016x-%016x.wal", &seq, &index)
	return seq, index, err
}

func walName(seq, index uint64) string {
	return fmt.Sprintf("%016x-%016x.wal", seq, index)
}
