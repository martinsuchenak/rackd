package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appLog "github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type LogService struct {
	store storage.ExtendedStorage
}

func NewLogService(store storage.ExtendedStorage) *LogService {
	return &LogService{store: store}
}

func (s *LogService) List(ctx context.Context, filter *model.LogFilter) ([]model.LogEntry, error) {
	if err := requirePermission(ctx, s.store, "logs", "list"); err != nil {
		return nil, err
	}
	return appLog.ListRecentEntries(filter), nil
}

func (s *LogService) Get(ctx context.Context, id string) (*model.LogEntry, error) {
	if err := requirePermission(ctx, s.store, "logs", "read"); err != nil {
		return nil, err
	}
	entry, ok := appLog.GetRecentEntry(id)
	if !ok {
		return nil, ErrNotFound
	}
	return entry, nil
}

func (s *LogService) Export(ctx context.Context, filter *model.LogFilter, format string) ([]byte, error) {
	if err := requirePermission(ctx, s.store, "logs", "export"); err != nil {
		return nil, err
	}

	entries := appLog.ListRecentEntries(filter)
	switch strings.ToLower(format) {
	case "csv":
		return exportLogEntriesCSV(entries), nil
	case "json", "":
		return json.MarshalIndent(entries, "", "  ")
	default:
		return nil, ValidationErrors{{Field: "format", Message: "format must be json or csv"}}
	}
}

func exportLogEntriesCSV(entries []model.LogEntry) []byte {
	var b strings.Builder
	b.WriteString("id,timestamp,level,source,message,fields\n")
	for _, entry := range entries {
		fields, _ := json.Marshal(entry.Fields)
		writeCSVCell(&b, entry.ID)
		b.WriteByte(',')
		writeCSVCell(&b, entry.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
		b.WriteByte(',')
		writeCSVCell(&b, entry.Level)
		b.WriteByte(',')
		writeCSVCell(&b, entry.Source)
		b.WriteByte(',')
		writeCSVCell(&b, entry.Message)
		b.WriteByte(',')
		writeCSVCell(&b, string(fields))
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func writeCSVCell(b *strings.Builder, value string) {
	escaped := strings.ReplaceAll(value, "\"", "\"\"")
	fmt.Fprintf(b, "\"%s\"", escaped)
}
