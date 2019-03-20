// +build !windows

package logutil

import (
	"bytes"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewJournalWriter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	jw, err := NewJournalWriter(buf)
	if err != nil {
		t.Skip(err)
	}

	syncer := zapcore.AddSync(jw)

	cr := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		syncer,
		zap.NewAtomicLevelAt(zap.InfoLevel),
	)

	lg := zap.New(cr, zap.AddCaller(), zap.ErrorOutput(syncer))
	defer lg.Sync()

	lg.Info("TestNewJournalWriter")
	if buf.String() == "" {
		// check with "journalctl -f"
		t.Log("sent logs successfully to journald")
	}
}
