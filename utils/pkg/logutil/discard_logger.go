package logutil

import (
	"log"

	"google.golang.org/grpc/grpclog"
)

// assert that "discardLogger" satisfy "Logger" interface
var _ Logger = &discardLogger{}

// NewDiscardLogger returns a new Logger that discards everything except "fatal".
func NewDiscardLogger() Logger { return &discardLogger{} }

type discardLogger struct{}

func (l *discardLogger) Info(args ...interface{})                    {}
func (l *discardLogger) Infoln(args ...interface{})                  {}
func (l *discardLogger) Infof(format string, args ...interface{})    {}
func (l *discardLogger) Warning(args ...interface{})                 {}
func (l *discardLogger) Warningln(args ...interface{})               {}
func (l *discardLogger) Warningf(format string, args ...interface{}) {}
func (l *discardLogger) Error(args ...interface{})                   {}
func (l *discardLogger) Errorln(args ...interface{})                 {}
func (l *discardLogger) Errorf(format string, args ...interface{})   {}
func (l *discardLogger) Fatal(args ...interface{})                   { log.Fatal(args...) }
func (l *discardLogger) Fatalln(args ...interface{})                 { log.Fatalln(args...) }
func (l *discardLogger) Fatalf(format string, args ...interface{})   { log.Fatalf(format, args...) }
func (l *discardLogger) V(lvl int) bool {
	return false
}
func (l *discardLogger) Lvl(lvl int) grpclog.LoggerV2 { return l }
