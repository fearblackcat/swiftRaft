// +build !windows

package logutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fearblackcat/smartRaft/utils/pkg/systemd"

	"github.com/coreos/go-systemd/journal"
	"go.uber.org/zap/zapcore"
)

// NewJournalWriter wraps "io.Writer" to redirect log output
// to the local systemd journal. If journald send fails, it fails
// back to writing to the original writer.
// The decode overhead is only <30µs per write.
// Reference: https://github.com/coreos/pkg/blob/master/capnslog/journald_formatter.go
func NewJournalWriter(wr io.Writer) (io.Writer, error) {
	return &journalWriter{Writer: wr}, systemd.DialJournal()
}

type journalWriter struct {
	io.Writer
}

// WARN: assume that etcd uses default field names in zap encoder config
// make sure to keep this up-to-date!
type logLine struct {
	Level  string `json:"level"`
	Caller string `json:"caller"`
}

func (w *journalWriter) Write(p []byte) (int, error) {
	line := &logLine{}
	if err := json.NewDecoder(bytes.NewReader(p)).Decode(line); err != nil {
		return 0, err
	}

	var pri journal.Priority
	switch line.Level {
	case zapcore.DebugLevel.String():
		pri = journal.PriDebug
	case zapcore.InfoLevel.String():
		pri = journal.PriInfo

	case zapcore.WarnLevel.String():
		pri = journal.PriWarning
	case zapcore.ErrorLevel.String():
		pri = journal.PriErr

	case zapcore.DPanicLevel.String():
		pri = journal.PriCrit
	case zapcore.PanicLevel.String():
		pri = journal.PriCrit
	case zapcore.FatalLevel.String():
		pri = journal.PriCrit

	default:
		panic(fmt.Errorf("unknown log level: %q", line.Level))
	}

	err := journal.Send(string(p), pri, map[string]string{
		"PACKAGE":           filepath.Dir(line.Caller),
		"SYSLOG_IDENTIFIER": filepath.Base(os.Args[0]),
	})
	if err != nil {
		// "journal" also falls back to stderr
		// "fmt.Fprintln(os.Stderr, s)"
		return w.Writer.Write(p)
	}
	return 0, nil
}
