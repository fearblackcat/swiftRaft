// Package pbutil defines interfaces for handling Protocol Buffer objects.
package pbutil

import "github.com/coreos/pkg/capnslog"

var (
	plog = capnslog.NewPackageLogger("git.xiaojukeji.com/gulfstream/dcron", "pkg/pbutil")
)

type Marshaler interface {
	Marshal() (data []byte, err error)
}

type Unmarshaler interface {
	Unmarshal(data []byte) error
}

func MustMarshal(m Marshaler) []byte {
	d, err := m.Marshal()
	if err != nil {
		plog.Panicf("marshal should never fail (%v)", err)
	}
	return d
}

func MustUnmarshal(um Unmarshaler, data []byte) {
	if err := um.Unmarshal(data); err != nil {
		plog.Panicf("unmarshal should never fail (%v)", err)
	}
}

func MaybeUnmarshal(um Unmarshaler, data []byte) bool {
	if err := um.Unmarshal(data); err != nil {
		return false
	}
	return true
}

func GetBool(v *bool) (vv bool, set bool) {
	if v == nil {
		return false, false
	}
	return *v, true
}

func Boolp(b bool) *bool { return &b }
