package logutil

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/coreos/pkg/capnslog"
)

func TestMergeLogger(t *testing.T) {
	var (
		txt      = "hello"
		repeatN  = 6
		duration = 2049843762 * time.Nanosecond
		mg       = NewMergeLogger(capnslog.NewPackageLogger("github.com/fearblackcat/swiftRaft", "utils/pkg/logutil"))
	)
	// overwrite this for testing
	defaultMergePeriod = time.Minute

	for i := 0; i < repeatN; i++ {
		mg.MergeError(txt)
		if i == 0 {
			time.Sleep(duration)
		}
	}

	if len(mg.statusm) != 1 {
		t.Errorf("got = %d, want = %d", len(mg.statusm), 1)
	}

	var l line
	for k := range mg.statusm {
		l = k
		break
	}

	if l.level != capnslog.ERROR {
		t.Errorf("got = %v, want = %v", l.level, capnslog.DEBUG)
	}
	if l.str != txt {
		t.Errorf("got = %s, want = %s", l.str, txt)
	}
	if mg.statusm[l].count != repeatN-1 {
		t.Errorf("got = %d, want = %d", mg.statusm[l].count, repeatN-1)
	}
	sum := mg.statusm[l].summary(time.Now())
	pre := fmt.Sprintf("[merged %d repeated lines in ", repeatN-1)
	if !strings.HasPrefix(sum, pre) {
		t.Errorf("got = %s, want = %s...", sum, pre)
	}
}
