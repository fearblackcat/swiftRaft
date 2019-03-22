package logutil

import (
	"github.com/coreos/pkg/capnslog"
	"google.golang.org/grpc/grpclog"
)

// assert that "packageLogger" satisfy "Logger" interface
var _ Logger = &packageLogger{}

// NewPackageLogger wraps "*capnslog.PackageLogger" that implements "Logger" interface.
//
// For example:
//
//  var defaultLogger Logger
//  defaultLogger = NewPackageLogger("github.com/fearblackcat/swiftRaft", "snapshot")
//
func NewPackageLogger(repo, pkg string) Logger {
	return &packageLogger{p: capnslog.NewPackageLogger(repo, pkg)}
}

type packageLogger struct {
	p *capnslog.PackageLogger
}

func (l *packageLogger) Info(args ...interface{})                    { l.p.Info(args...) }
func (l *packageLogger) Infoln(args ...interface{})                  { l.p.Info(args...) }
func (l *packageLogger) Infof(format string, args ...interface{})    { l.p.Infof(format, args...) }
func (l *packageLogger) Warning(args ...interface{})                 { l.p.Warning(args...) }
func (l *packageLogger) Warningln(args ...interface{})               { l.p.Warning(args...) }
func (l *packageLogger) Warningf(format string, args ...interface{}) { l.p.Warningf(format, args...) }
func (l *packageLogger) Error(args ...interface{})                   { l.p.Error(args...) }
func (l *packageLogger) Errorln(args ...interface{})                 { l.p.Error(args...) }
func (l *packageLogger) Errorf(format string, args ...interface{})   { l.p.Errorf(format, args...) }
func (l *packageLogger) Fatal(args ...interface{})                   { l.p.Fatal(args...) }
func (l *packageLogger) Fatalln(args ...interface{})                 { l.p.Fatal(args...) }
func (l *packageLogger) Fatalf(format string, args ...interface{})   { l.p.Fatalf(format, args...) }
func (l *packageLogger) V(lvl int) bool {
	return l.p.LevelAt(capnslog.LogLevel(lvl))
}
func (l *packageLogger) Lvl(lvl int) grpclog.LoggerV2 {
	if l.p.LevelAt(capnslog.LogLevel(lvl)) {
		return l
	}
	return &discardLogger{}
}
