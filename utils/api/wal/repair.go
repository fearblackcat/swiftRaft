package wal

import (
	"io"
	"os"
	"path/filepath"

	"git.xiaojukeji.com/gulfstream/dcron/workflow/logtool"
	"github.com/fearblackcat/smartRaft/utils/api/wal/walpb"
	"github.com/fearblackcat/smartRaft/utils/pkg/fileutil"
)

// Repair tries to repair ErrUnexpectedEOF in the
// last wal file by truncating.
func Repair(lg *logtool.RLogHandle, dirpath string) bool {
	f, err := openLast(lg, dirpath)
	if err != nil {
		return false
	}
	defer f.Close()

	if lg != nil {
		lg.Info("repairing", map[string]interface{}{
			"path": f.Name(),
		})
	} else {
		plog.Noticef("repairing %v", f.Name())
	}

	rec := &walpb.Record{}
	decoder := newDecoder(f)
	for {
		lastOffset := decoder.lastOffset()
		err := decoder.decode(rec)
		switch err {
		case nil:
			// update crc of the decoder when necessary
			switch rec.Type {
			case crcType:
				crc := decoder.crc.Sum32()
				// current crc of decoder must match the crc of the record.
				// do no need to match 0 crc, since the decoder is a new one at this case.
				if crc != 0 && rec.Validate(crc) != nil {
					return false
				}
				decoder.updateCRC(rec.Crc)
			}
			continue

		case io.EOF:
			if lg != nil {
				lg.Info("repaired", map[string]interface{}{
					"path":  f.Name(),
					"error": io.EOF,
				})
			}
			return true

		case io.ErrUnexpectedEOF:
			bf, bferr := os.Create(f.Name() + ".broken")
			if bferr != nil {
				if lg != nil {
					lg.Warn("failed to create backup file", map[string]interface{}{
						"path": f.Name() + ".broken",
					})
				} else {
					plog.Errorf("could not repair %v, failed to create backup file", f.Name())
				}
				return false
			}
			defer bf.Close()

			if _, err = f.Seek(0, io.SeekStart); err != nil {
				if lg != nil {
					lg.Warn("failed to read file", map[string]interface{}{
						"path":  f.Name(),
						"error": err,
					})
				} else {
					plog.Errorf("could not repair %v, failed to read file", f.Name())
				}
				return false
			}

			if _, err = io.Copy(bf, f); err != nil {
				if lg != nil {
					lg.Warn("failed to copy", map[string]interface{}{
						"from":  f.Name() + ".broken",
						"to":    f.Name(),
						"error": err,
					})
				} else {
					plog.Errorf("could not repair %v, failed to copy file", f.Name())
				}
				return false
			}

			if err = f.Truncate(lastOffset); err != nil {
				if lg != nil {
					lg.Warn("failed to truncate", map[string]interface{}{
						"path":  f.Name(),
						"error": err,
					})
				} else {
					plog.Errorf("could not repair %v, failed to truncate file", f.Name())
				}
				return false
			}

			if err = fileutil.Fsync(f.File); err != nil {
				if lg != nil {
					lg.Warn("failed to fsync", map[string]interface{}{
						"path":  f.Name(),
						"error": err,
					})
				} else {
					plog.Errorf("could not repair %v, failed to sync file", f.Name())
				}
				return false
			}

			if lg != nil {
				lg.Info("repaired", map[string]interface{}{
					"path":  f.Name(),
					"error": io.ErrUnexpectedEOF,
				})
			}
			return true

		default:
			if lg != nil {
				lg.Warn("failed to repair", map[string]interface{}{
					"path":  f.Name(),
					"error": err,
				})
			} else {
				plog.Errorf("could not repair error (%v)", err)
			}
			return false
		}
	}
}

// openLast opens the last wal file for read and write.
func openLast(lg *logtool.RLogHandle, dirpath string) (*fileutil.LockedFile, error) {
	names, err := readWALNames(lg, dirpath)
	if err != nil {
		return nil, err
	}
	last := filepath.Join(dirpath, names[len(names)-1])
	return fileutil.LockFile(last, os.O_RDWR, fileutil.PrivateFileMode)
}
