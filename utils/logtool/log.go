package logtool

import (
	"github.com/Sirupsen/logrus"
)

type DLTag string

var (
	RLog = &RLogHandle{logrus.NewEntry(logrus.New())}
	NLog = &NLogHandle{logrus.NewEntry(logrus.New())}
)

type RLogHandle struct {
	*logrus.Entry
}

// Debug logs a message at DebugLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (rLog *RLogHandle) Debug(msg string, fields map[string]interface{}) {
	rLog.Entry.WithFields(logrus.Fields(fields)).Debug(msg)
}

// Info logs a message at InfoLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (rLog *RLogHandle) Info(msg string, fields map[string]interface{}) {
	rLog.Entry.WithFields(logrus.Fields(fields)).Info(msg)
}

// Warn logs a message at WarnLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (rLog *RLogHandle) Warn(msg string, fields map[string]interface{}) {
	rLog.Entry.WithFields(logrus.Fields(fields)).Warn(msg)
}

// Error logs a message at ErrorLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (rLog *RLogHandle) Error(msg string, fields map[string]interface{}) {
	rLog.Entry.WithFields(logrus.Fields(fields)).Error(msg)
}

// Panic logs a message at PanicLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then panics, even if logging at PanicLevel is disabled.
func (rLog *RLogHandle) Panic(msg string, fields map[string]interface{}) {
	rLog.Entry.WithFields(logrus.Fields(fields)).Panic(msg)
}

// Fatal logs a message at FatalLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then calls os.Exit(1), even if logging at FatalLevel is
// disabled.
func (rLog *RLogHandle) Fatal(msg string, fields map[string]interface{}) {
	rLog.Entry.WithFields(logrus.Fields(fields)).Fatal(msg)
}

type NLogHandle struct {
	*logrus.Entry
}

func (nLog *NLogHandle) Debug(v ...interface{}) {
	nLog.Entry.Debug(v)
}

func (nLog *NLogHandle) Debugf(format string, v ...interface{}) {
	nLog.Entry.Debugf(format, v)
}

func (nLog *NLogHandle) Error(v ...interface{}) {
	nLog.Entry.Error(v)
}

func (nLog *NLogHandle) Errorf(format string, v ...interface{}) {
	nLog.Entry.Errorf(format, v)
}

func (nLog *NLogHandle) Info(v ...interface{}) {
	nLog.Entry.Info(v)
}

func (nLog *NLogHandle) Infof(format string, v ...interface{}) {
	nLog.Entry.Infof(format, v)
}

func (nLog *NLogHandle) Warning(v ...interface{}) {
	nLog.Entry.Warning(v)
}

func (nLog *NLogHandle) Warningf(format string, v ...interface{}) {
	nLog.Entry.Warningf(format, v)
}

func (nLog *NLogHandle) Fatal(v ...interface{}) {
	nLog.Entry.Fatal(v)
}

func (nLog *NLogHandle) Fatalf(format string, v ...interface{}) {
	nLog.Logger.Fatalf(format, v)
}

func (nLog *NLogHandle) Panic(v ...interface{}) {
	nLog.Entry.Panic(v)
}

func (nLog *NLogHandle) Panicf(format string, v ...interface{}) {
	nLog.Entry.Panicf(format, v)
}

// InitLogger creates the logger instance
func InitRaftLogger(logLevel string, node string) {
	formattedLogger := logrus.New()
	formattedLogger.Formatter = &logrus.TextFormatter{FullTimestamp: true}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.WithError(err).Error("Error parsing log level, using: info")
		level = logrus.InfoLevel
	}

	formattedLogger.Level = level
	logger := logrus.NewEntry(formattedLogger).WithField("node", node)

	RLog = &RLogHandle{logger}
}

func InitNodeMsgLogger(logLevel string, node string) {
	formattedLogger := logrus.New()
	formattedLogger.Formatter = &logrus.TextFormatter{FullTimestamp: true}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.WithError(err).Error("Error parsing log level, using: info")
		level = logrus.InfoLevel
	}

	formattedLogger.Level = level
	logger := logrus.NewEntry(formattedLogger).WithField("node", node)

	NLog = &NLogHandle{logger}
}
