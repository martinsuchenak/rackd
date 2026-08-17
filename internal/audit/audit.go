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
		buf.WriteString(csvCell(log.ID))
		buf.WriteByte(',')
		buf.WriteString(log.Timestamp.Format(time.RFC3339))
		buf.WriteByte(',')
		buf.WriteString(csvCell(log.Action))
		buf.WriteByte(',')
		buf.WriteString(csvCell(log.Resource))
		buf.WriteByte(',')
		buf.WriteString(csvCell(log.ResourceID))
		buf.WriteByte(',')
		buf.WriteString(csvCell(log.UserID))
		buf.WriteByte(',')
		buf.WriteString(csvCell(log.Username))
		buf.WriteByte(',')
		buf.WriteString(csvCell(log.IPAddress))
		buf.WriteByte(',')
		buf.WriteString(csvCell(log.Status))
		buf.WriteByte(',')
		buf.WriteString(csvCell(log.Error))
		buf.WriteByte(',')
		buf.WriteString(csvCell(log.Source))
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

// csvCell escapes a value for CSV and neutralizes CSV formula injection:
// values that start with =, +, -, @, tab or CR would be interpreted as
// formulas by Excel/Sheets when an admin opens an export containing
// attacker-influenced text (OWASP guidance).
func csvCell(s string) string {
	switch {
	case s == "":
		return s
	case s[0] == '=', s[0] == '+', s[0] == '-', s[0] == '@', s[0] == '\t', s[0] == '\r':
		s = "'" + s
	}
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
