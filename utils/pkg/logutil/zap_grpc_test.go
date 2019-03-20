package logutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewGRPCLoggerV2(t *testing.T) {
	logPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-log-%d", time.Now().UnixNano()))
	defer os.RemoveAll(logPath)

	lcfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{logPath},
		ErrorOutputPaths: []string{logPath},
	}
	gl, err := NewGRPCLoggerV2(lcfg)
	if err != nil {
		t.Fatal(err)
	}

	// debug level is not enabled,
	// so info level gRPC-side logging is discarded
	gl.Info("etcd-logutil-1")
	data, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("etcd-logutil-1")) {
		t.Fatalf("unexpected line %q", string(data))
	}

	gl.Warning("etcd-logutil-2")
	data, err = ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("etcd-logutil-2")) {
		t.Fatalf("can't find data in log %q", string(data))
	}
	if !bytes.Contains(data, []byte("logutil/zap_grpc_test.go:")) {
		t.Fatalf("unexpected caller; %q", string(data))
	}
}

func TestNewGRPCLoggerV2FromZapCore(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	syncer := zapcore.AddSync(buf)
	cr := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		syncer,
		zap.NewAtomicLevelAt(zap.InfoLevel),
	)

	lg := NewGRPCLoggerV2FromZapCore(cr, syncer)
	lg.Warning("TestNewGRPCLoggerV2FromZapCore")
	txt := buf.String()
	if !strings.Contains(txt, "TestNewGRPCLoggerV2FromZapCore") {
		t.Fatalf("unexpected log %q", txt)
	}
}
