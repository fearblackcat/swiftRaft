package snap

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fearblackcat/swiftRaft/raft"
	"github.com/fearblackcat/swiftRaft/raft/raftpb"
	"github.com/fearblackcat/swiftRaft/utils/api/snap/snappb"
	"github.com/fearblackcat/swiftRaft/utils/logtool"
	pioutil "github.com/fearblackcat/swiftRaft/utils/pkg/ioutil"
	"github.com/fearblackcat/swiftRaft/utils/pkg/pbutil"

	"github.com/coreos/pkg/capnslog"
)

const snapSuffix = ".snap"

var (
	plog = capnslog.NewPackageLogger("github.com/fearblackcat/swiftRaft", "snap")

	ErrNoSnapshot    = errors.New("snap: no available snapshot")
	ErrEmptySnapshot = errors.New("snap: empty snapshot")
	ErrCRCMismatch   = errors.New("snap: crc mismatch")
	crcTable         = crc32.MakeTable(crc32.Castagnoli)

	// A map of valid files that can be present in the snap folder.
	validFiles = map[string]bool{
		"db": true,
	}
)

type Snapshotter struct {
	lg  *logtool.RLogHandle
	dir string
}

func New(lg *logtool.RLogHandle, dir string) *Snapshotter {
	return &Snapshotter{
		lg:  lg,
		dir: dir,
	}
}

func (s *Snapshotter) SaveSnap(snapshot raftpb.Snapshot) error {
	if raft.IsEmptySnap(snapshot) {
		return nil
	}
	return s.save(&snapshot)
}

func (s *Snapshotter) save(snapshot *raftpb.Snapshot) error {
	start := time.Now()

	fname := fmt.Sprintf("%016x-%016x%s", snapshot.Metadata.Term, snapshot.Metadata.Index, snapSuffix)
	b := pbutil.MustMarshal(snapshot)
	crc := crc32.Update(0, crcTable, b)
	snap := snappb.Snapshot{Crc: crc, Data: b}
	d, err := snap.Marshal()
	if err != nil {
		return err
	}
	snapMarshallingSec.Observe(time.Since(start).Seconds())

	spath := filepath.Join(s.dir, fname)

	fsyncStart := time.Now()
	err = pioutil.WriteAndSyncFile(spath, d, 0666)
	snapFsyncSec.Observe(time.Since(fsyncStart).Seconds())

	if err != nil {
		if s.lg != nil {
			s.lg.Warn("failed to write a snap file", map[string]interface{}{
				"path":  spath,
				"error": err,
			})
		}
		rerr := os.Remove(spath)
		if rerr != nil {
			if s.lg != nil {
				s.lg.Warn("failed to remove a broken snap file", map[string]interface{}{
					"path":  spath,
					"error": err,
				})
			} else {
				plog.Errorf("failed to remove broken snapshot file %s", spath)
			}
		}
		return err
	}

	snapSaveSec.Observe(time.Since(start).Seconds())
	return nil
}

func (s *Snapshotter) Load() (*raftpb.Snapshot, error) {
	names, err := s.snapNames()
	if err != nil {
		return nil, err
	}
	var snap *raftpb.Snapshot
	for _, name := range names {
		if snap, err = loadSnap(s.lg, s.dir, name); err == nil {
			break
		}
	}
	if err != nil {
		return nil, ErrNoSnapshot
	}
	return snap, nil
}

func loadSnap(lg *logtool.RLogHandle, dir, name string) (*raftpb.Snapshot, error) {
	fpath := filepath.Join(dir, name)
	snap, err := Read(lg, fpath)
	if err != nil {
		brokenPath := fpath + ".broken"
		if lg != nil {
			lg.Warn("failed to read a snap file", map[string]interface{}{
				"path":  fpath,
				"error": err,
			})
		}
		if rerr := os.Rename(fpath, brokenPath); rerr != nil {
			if lg != nil {
				lg.Warn("failed to rename a broken snap file", map[string]interface{}{
					"path":        fpath,
					"broken-path": brokenPath,
					"error":       rerr,
				})
			} else {
				plog.Warningf("cannot rename broken snapshot file %v to %v: %v", fpath, brokenPath, rerr)
			}
		} else {
			if lg != nil {
				lg.Warn("renamed to a broken snap file", map[string]interface{}{
					"path":        fpath,
					"broken-path": brokenPath,
				})
			}
		}
	}
	return snap, err
}

// Read reads the snapshot named by snapname and returns the snapshot.
func Read(lg *logtool.RLogHandle, snapname string) (*raftpb.Snapshot, error) {
	b, err := ioutil.ReadFile(snapname)
	if err != nil {
		if lg != nil {
			lg.Warn("failed to read a snap file", map[string]interface{}{
				"path":  snapname,
				"error": err,
			})
		} else {
			plog.Errorf("cannot read file %v: %v", snapname, err)
		}
		return nil, err
	}

	if len(b) == 0 {
		if lg != nil {
			lg.Warn("failed to read empty snapshot file", map[string]interface{}{
				"path": snapname,
			})
		} else {
			plog.Errorf("unexpected empty snapshot")
		}
		return nil, ErrEmptySnapshot
	}

	var serializedSnap snappb.Snapshot
	if err = serializedSnap.Unmarshal(b); err != nil {
		if lg != nil {
			lg.Warn("failed to unmarshal snappb.Snapshot", map[string]interface{}{
				"path":  snapname,
				"error": err,
			})
		} else {
			plog.Errorf("corrupted snapshot file %v: %v", snapname, err)
		}
		return nil, err
	}

	if len(serializedSnap.Data) == 0 || serializedSnap.Crc == 0 {
		if lg != nil {
			lg.Warn("failed to read empty snapshot data", map[string]interface{}{
				"path": snapname,
			})
		} else {
			plog.Errorf("unexpected empty snapshot")
		}
		return nil, ErrEmptySnapshot
	}

	crc := crc32.Update(0, crcTable, serializedSnap.Data)
	if crc != serializedSnap.Crc {
		if lg != nil {
			lg.Warn("snap file is corrupt", map[string]interface{}{
				"path":    snapname,
				"pre-crc": serializedSnap.Crc,
				"new-crc": crc,
			})
		} else {
			plog.Errorf("corrupted snapshot file %v: crc mismatch", snapname)
		}
		return nil, ErrCRCMismatch
	}

	var snap raftpb.Snapshot
	if err = snap.Unmarshal(serializedSnap.Data); err != nil {
		if lg != nil {
			lg.Warn("failed to unmarshal raftpb.Snapshot", map[string]interface{}{
				"path":  snapname,
				"error": err,
			})
		} else {
			plog.Errorf("corrupted snapshot file %v: %v", snapname, err)
		}
		return nil, err
	}
	return &snap, nil
}

// snapNames returns the filename of the snapshots in logical time order (from newest to oldest).
// If there is no available snapshots, an ErrNoSnapshot will be returned.
func (s *Snapshotter) snapNames() ([]string, error) {
	dir, err := os.Open(s.dir)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	snaps := checkSuffix(s.lg, names)
	if len(snaps) == 0 {
		return nil, ErrNoSnapshot
	}
	sort.Sort(sort.Reverse(sort.StringSlice(snaps)))
	return snaps, nil
}

func checkSuffix(lg *logtool.RLogHandle, names []string) []string {
	snaps := []string{}
	for i := range names {
		if strings.HasSuffix(names[i], snapSuffix) {
			snaps = append(snaps, names[i])
		} else {
			// If we find a file which is not a snapshot then check if it's
			// a vaild file. If not throw out a warning.
			if _, ok := validFiles[names[i]]; !ok {
				if lg != nil {
					lg.Warn("found unexpected non-snap file; skipping", map[string]interface{}{
						"path": names[i],
					})
				} else {
					plog.Warningf("skipped unexpected non snapshot file %v", names[i])
				}
			}
		}
	}
	return snaps
}
