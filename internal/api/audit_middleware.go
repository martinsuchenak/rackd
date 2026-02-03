package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

// AuditMiddleware logs all API changes to the audit log
func AuditMiddleware(store storage.AuditStorage) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip non-mutating operations and health/metrics endpoints
			if !shouldAudit(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Capture request body for changes tracking
			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// Wrap response writer to capture status
			wrapped := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log to audit trail
			go logAudit(store, r, wrapped.statusCode, bodyBytes)
		})
	}
}

type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *auditResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func shouldAudit(r *http.Request) bool {
	// Only audit mutating operations
	if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
		return false
	}

	// Skip health, metrics, and static endpoints
	path := r.URL.Path
	skipPaths := []string{"/healthz", "/readyz", "/metrics", "/mcp", "/static/", "/api/search"}
	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return false
		}
	}

	return strings.HasPrefix(path, "/api/")
}

func logAudit(store storage.AuditStorage, r *http.Request, statusCode int, bodyBytes []byte) {
	action, resource, resourceID := parseRequest(r)

	auditLog := &model.AuditLog{
		Timestamp:  time.Now(),
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  getClientIP(r),
		Status:     getStatus(statusCode),
		Source:     "api",
	}

	// Extract user info from context (if authenticated)
	if apiKey, ok := r.Context().Value(APIKeyContextKey).(*model.APIKey); ok {
		auditLog.UserID = apiKey.ID
		auditLog.Username = apiKey.Name
	}

	// Store changes as JSON
	if len(bodyBytes) > 0 && len(bodyBytes) < 10000 { // Limit size
		auditLog.Changes = string(bodyBytes)
	}

	if err := store.CreateAuditLog(auditLog); err != nil {
		log.Error("Failed to create audit log", "error", err)
	}
}

func parseRequest(r *http.Request) (action, resource, resourceID string) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(path, "/")

	// Determine action from method
	switch r.Method {
	case "POST":
		action = "create"
	case "PUT", "PATCH":
		action = "update"
	case "DELETE":
		action = "delete"
	default:
		action = r.Method
	}

	// Extract resource and ID
	if len(parts) > 0 {
		resource = parts[0]
		if len(parts) > 1 && parts[1] != "bulk" {
			resourceID = parts[1]
		}
	}

	// Handle bulk operations
	if len(parts) > 1 && parts[1] == "bulk" {
		action = "bulk_" + action
	}

	return
}

func getStatus(statusCode int) string {
	if statusCode >= 200 && statusCode < 300 {
		return "success"
	}
	return "failure"
}

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
