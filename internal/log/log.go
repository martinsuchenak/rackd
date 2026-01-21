package log

import (
	"io"
	"os"

	"github.com/paularlott/logger"
	logslog "github.com/paularlott/logger/slog"
)

var defaultLogger logger.Logger

func Init(logFormat string, logLevel string, writer io.Writer) {
	if writer == nil {
		writer = os.Stdout
	}

	if logFormat == "" {
		logFormat = "console"
	}
	if logLevel == "" {
		logLevel = "info"
	}

	defaultLogger = logslog.New(logslog.Config{
		Level:  logLevel,
		Format: logFormat,
		Writer: writer,
	})
}

func Trace(msg string, keysAndValues ...any) {
	defaultLogger.Trace(msg, keysAndValues...)
}

func Debug(msg string, keysAndValues ...any) {
	defaultLogger.Debug(msg, keysAndValues...)
}

func Info(msg string, keysAndValues ...any) {
	defaultLogger.Info(msg, keysAndValues...)
}

func Warn(msg string, keysAndValues ...any) {
	defaultLogger.Warn(msg, keysAndValues...)
}

func Error(msg string, keysAndValues ...any) {
	defaultLogger.Error(msg, keysAndValues...)
}

func Fatal(msg string, keysAndValues ...any) {
	defaultLogger.Fatal(msg, keysAndValues...)
}

func With(key string, value any) logger.Logger {
	return defaultLogger.With(key, value)
}

func WithError(err error) logger.Logger {
	return defaultLogger.WithError(err)
}

func WithGroup(group string) logger.Logger {
	return defaultLogger.WithGroup(group)
}

func GetLogger() logger.Logger {
	return defaultLogger
}
