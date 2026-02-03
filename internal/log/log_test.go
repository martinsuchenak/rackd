package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name      string
		logFormat string
		logLevel  string
	}{
		{"default", "", ""},
		{"console info", "console", "info"},
		{"json debug", "json", "debug"},
		{"console error", "console", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			Init(tt.logFormat, tt.logLevel, buf)
			if defaultLogger == nil {
				t.Error("defaultLogger should not be nil after Init")
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	Init("console", "trace", buf)

	tests := []struct {
		name string
		fn   func(string, ...any)
		msg  string
	}{
		{"trace", Trace, "trace message"},
		{"debug", Debug, "debug message"},
		{"info", Info, "info message"},
		{"warn", Warn, "warn message"},
		{"error", Error, "error message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.fn(tt.msg)
			output := buf.String()
			if !strings.Contains(output, tt.msg) {
				t.Errorf("expected output to contain %q, got %q", tt.msg, output)
			}
		})
	}
}

func TestLogWithKeyValues(t *testing.T) {
	buf := &bytes.Buffer{}
	Init("json", "info", buf)

	Info("test message", "key1", "value1", "key2", 42)
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Error("expected output to contain message")
	}
}

func TestWith(t *testing.T) {
	buf := &bytes.Buffer{}
	Init("console", "info", buf)

	logger := With("component", "test")
	if logger == nil {
		t.Error("With() should return a logger")
	}
}

func TestWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	Init("console", "info", buf)

	err := &testError{"test error"}
	logger := WithError(err)
	if logger == nil {
		t.Error("WithError() should return a logger")
	}
}

func TestWithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	Init("console", "info", buf)

	logger := WithGroup("testgroup")
	if logger == nil {
		t.Error("WithGroup() should return a logger")
	}
}

func TestGetLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	Init("console", "info", buf)

	logger := GetLogger()
	if logger == nil {
		t.Error("GetLogger() should return a logger")
	}
	if logger != defaultLogger {
		t.Error("GetLogger() should return the default logger")
	}
}

func TestInitWithNilWriter(t *testing.T) {
	Init("console", "info", nil)
	if defaultLogger == nil {
		t.Error("Init with nil writer should still initialize logger")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
