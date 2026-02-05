package audit

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Export audit logs to JSON
func ExportAuditLogsJSON(logs []model.AuditLog) ([]byte, error) {
	return json.MarshalIndent(logs, "", "  ")
}

// Export audit logs to CSV
func ExportAuditLogsCSV(logs []model.AuditLog) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("ID,Timestamp,Action,Resource,ResourceID,UserID,Username,IPAddress,Status,Error,Source\n")

	for _, log := range logs {
		buf.WriteString(csvEscape(log.ID))
		buf.WriteByte(',')
		buf.WriteString(log.Timestamp.Format(time.RFC3339))
		buf.WriteByte(',')
		buf.WriteString(csvEscape(log.Action))
		buf.WriteByte(',')
		buf.WriteString(csvEscape(log.Resource))
		buf.WriteByte(',')
		buf.WriteString(csvEscape(log.ResourceID))
		buf.WriteByte(',')
		buf.WriteString(csvEscape(log.UserID))
		buf.WriteByte(',')
		buf.WriteString(csvEscape(log.Username))
		buf.WriteByte(',')
		buf.WriteString(csvEscape(log.IPAddress))
		buf.WriteByte(',')
		buf.WriteString(csvEscape(log.Status))
		buf.WriteByte(',')
		buf.WriteString(csvEscape(log.Error))
		buf.WriteByte(',')
		buf.WriteString(csvEscape(log.Source))
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
