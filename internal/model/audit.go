package model

import "time"

// AuditLog represents a single audit log entry
type AuditLog struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Action     string    `json:"action"`      // create, update, delete, login, etc.
	Resource   string    `json:"resource"`    // device, network, datacenter, etc.
	ResourceID string    `json:"resource_id"` // ID of the affected resource
	UserID     string    `json:"user_id"`     // API key ID or user ID
	Username   string    `json:"username"`    // API key name or username
	IPAddress  string    `json:"ip_address"`  // Client IP
	Changes    string    `json:"changes"`     // JSON of changes (before/after)
	Status     string    `json:"status"`      // success, failure
	Error      string    `json:"error"`       // Error message if failed
	Source     string    `json:"source"`      // Entry point: api, mcp, cli, discovery, scheduler
}

// AuditFilter for querying audit logs
type AuditFilter struct {
	Pagination
	Resource   string
	ResourceID string
	UserID     string
	Action     string
	Source     string
	StartTime  *time.Time
	EndTime    *time.Time
}
