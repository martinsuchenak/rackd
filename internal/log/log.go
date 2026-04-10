package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/paularlott/logger"
	logslog "github.com/paularlott/logger/slog"
)

var defaultLogger logger.Logger

const recentLogCapacity = 2000

type capturingLogger struct {
	inner  logger.Logger
	fields map[string]string
	group  string
}

type recentLogStore struct {
	mu      sync.RWMutex
	entries []model.LogEntry
}

var recentLogs = &recentLogStore{
	entries: make([]model.LogEntry, 0, recentLogCapacity),
}

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

	recentLogs.clear()
	base := logslog.New(logslog.Config{
		Level:  logLevel,
		Format: logFormat,
		Writer: writer,
	})
	defaultLogger = &capturingLogger{
		inner:  base,
		fields: map[string]string{},
	}
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

func ListRecentEntries(filter *model.LogFilter) []model.LogEntry {
	return recentLogs.list(filter)
}

func GetRecentEntry(id string) (*model.LogEntry, bool) {
	return recentLogs.get(id)
}

func ClearRecentEntries() {
	recentLogs.clear()
}

func (l *capturingLogger) Trace(msg string, keysAndValues ...any) {
	l.capture("trace", msg, keysAndValues...)
	l.inner.Trace(msg, keysAndValues...)
}

func (l *capturingLogger) Debug(msg string, keysAndValues ...any) {
	l.capture("debug", msg, keysAndValues...)
	l.inner.Debug(msg, keysAndValues...)
}

func (l *capturingLogger) Info(msg string, keysAndValues ...any) {
	l.capture("info", msg, keysAndValues...)
	l.inner.Info(msg, keysAndValues...)
}

func (l *capturingLogger) Warn(msg string, keysAndValues ...any) {
	l.capture("warn", msg, keysAndValues...)
	l.inner.Warn(msg, keysAndValues...)
}

func (l *capturingLogger) Error(msg string, keysAndValues ...any) {
	l.capture("error", msg, keysAndValues...)
	l.inner.Error(msg, keysAndValues...)
}

func (l *capturingLogger) Fatal(msg string, keysAndValues ...any) {
	l.capture("fatal", msg, keysAndValues...)
	l.inner.Fatal(msg, keysAndValues...)
}

func (l *capturingLogger) With(key string, value any) logger.Logger {
	fields := cloneFields(l.fields)
	fields[key] = sanitizeField(key, stringifyValue(value))
	return &capturingLogger{
		inner:  l.inner.With(key, value),
		fields: fields,
		group:  l.group,
	}
}

func (l *capturingLogger) WithError(err error) logger.Logger {
	fields := cloneFields(l.fields)
	fields["error"] = sanitizeField("error", stringifyValue(err))
	return &capturingLogger{
		inner:  l.inner.WithError(err),
		fields: fields,
		group:  l.group,
	}
}

func (l *capturingLogger) WithGroup(group string) logger.Logger {
	nextGroup := group
	if l.group != "" {
		nextGroup = l.group + "." + group
	}
	return &capturingLogger{
		inner:  l.inner.WithGroup(group),
		fields: cloneFields(l.fields),
		group:  nextGroup,
	}
}

func (l *capturingLogger) capture(level, msg string, keysAndValues ...any) {
	fields := cloneFields(l.fields)
	for i := 0; i < len(keysAndValues); i += 2 {
		key := fmt.Sprint(keysAndValues[i])
		value := ""
		if i+1 < len(keysAndValues) {
			value = stringifyValue(keysAndValues[i+1])
		}
		fields[key] = sanitizeField(key, value)
	}

	source := fields["source"]
	if source == "" {
		source = fields["component"]
	}
	if source == "" {
		source = l.group
	}

	recentLogs.add(model.LogEntry{
		ID:        uuid.Must(uuid.NewV7()).String(),
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   msg,
		Source:    source,
		Fields:    fields,
	})
}

func cloneFields(fields map[string]string) map[string]string {
	cloned := make(map[string]string, len(fields))
	for key, value := range fields {
		cloned[key] = value
	}
	return cloned
}

func stringifyValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case error:
		return typed.Error()
	case string:
		return typed
	default:
		if encoded, err := json.Marshal(typed); err == nil && string(encoded) != "null" {
			return string(encoded)
		}
		return fmt.Sprint(typed)
	}
}

func sanitizeField(key, value string) string {
	lower := strings.ToLower(key)
	sensitiveKeys := []string{
		"password",
		"secret",
		"token",
		"authorization",
		"api_key",
		"apikey",
		"community",
		"private_key",
		"provider_key",
		"auth",
		"priv",
	}
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(lower, sensitive) {
			return "[redacted]"
		}
	}
	return value
}

func (s *recentLogStore) add(entry model.LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.entries) == recentLogCapacity {
		copy(s.entries, s.entries[1:])
		s.entries[len(s.entries)-1] = entry
		return
	}
	s.entries = append(s.entries, entry)
}

func (s *recentLogStore) list(filter *model.LogFilter) []model.LogEntry {
	s.mu.RLock()
	entries := make([]model.LogEntry, len(s.entries))
	copy(entries, s.entries)
	s.mu.RUnlock()

	filtered := make([]model.LogEntry, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if filter != nil {
			if filter.Level != "" && !strings.EqualFold(entry.Level, filter.Level) {
				continue
			}
			if filter.Source != "" && !strings.EqualFold(entry.Source, filter.Source) {
				continue
			}
			if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
				continue
			}
			if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
				continue
			}
			if filter.Query != "" && !matchesLogQuery(entry, filter.Query) {
				continue
			}
		}
		filtered = append(filtered, entry)
	}

	if filter == nil {
		return filtered
	}

	pagination := filter.Pagination
	pagination.Clamp()
	if pagination.Offset >= len(filtered) {
		return []model.LogEntry{}
	}
	end := pagination.Offset + pagination.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[pagination.Offset:end]
}

func matchesLogQuery(entry model.LogEntry, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(entry.Message), q) || strings.Contains(strings.ToLower(entry.Source), q) {
		return true
	}
	for key, value := range entry.Fields {
		if strings.Contains(strings.ToLower(key), q) || strings.Contains(strings.ToLower(value), q) {
			return true
		}
	}
	return false
}

func (s *recentLogStore) get(id string) (*model.LogEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].ID == id {
			entry := s.entries[i]
			return &entry, true
		}
	}
	return nil, false
}

func (s *recentLogStore) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = s.entries[:0]
}
