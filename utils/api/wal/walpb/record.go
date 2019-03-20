package walpb

import "errors"

var (
	ErrCRCMismatch = errors.New("walpb: crc mismatch")
)

func (rec *Record) Validate(crc uint32) error {
	if rec.Crc == crc {
		return nil
	}
	rec.Reset()
	return ErrCRCMismatch
}
