package logutil_test

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/fearblackcat/smartRaft/utils/pkg/logutil"

	"github.com/coreos/pkg/capnslog"
)

func TestPackageLogger(t *testing.T) {
	buf := new(bytes.Buffer)
	capnslog.SetFormatter(capnslog.NewDefaultFormatter(buf))

	l := logutil.NewPackageLogger("git.xiaojukeji.com/gulfstream/dcron", "logger")

	r := capnslog.MustRepoLogger("git.xiaojukeji.com/gulfstream/dcron")
	r.SetLogLevel(map[string]capnslog.LogLevel{"logger": capnslog.INFO})

	l.Infof("hello world!")
	if !strings.Contains(buf.String(), "hello world!") {
		t.Fatalf("expected 'hello world!', got %q", buf.String())
	}
	buf.Reset()

	// capnslog.INFO is 3
	l.Lvl(2).Infof("Level 2")
	l.Lvl(5).Infof("Level 5")
	if !strings.Contains(buf.String(), "Level 2") {
		t.Fatalf("expected 'Level 2', got %q", buf.String())
	}
	if strings.Contains(buf.String(), "Level 5") {
		t.Fatalf("unexpected 'Level 5', got %q", buf.String())
	}
	buf.Reset()

	capnslog.SetFormatter(capnslog.NewDefaultFormatter(ioutil.Discard))
	l.Infof("ignore this")
	if len(buf.Bytes()) > 0 {
		t.Fatalf("unexpected logs %q", buf.String())
	}
}
