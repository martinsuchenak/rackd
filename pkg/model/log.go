package model

import "time"

// LogEntry represents a recent in-app log entry exposed to administrators.
type LogEntry struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Source    string            `json:"source,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// LogFilter provides filtering and pagination for recent log queries.
type LogFilter struct {
	Pagination
	Level     string
	Source    string
	Query     string
	StartTime *time.Time
	EndTime   *time.Time
}
