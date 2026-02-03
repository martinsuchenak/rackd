package storage

import (
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestAuditLog(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	// Create audit log
	log := &model.AuditLog{
		Action:     "create",
		Resource:   "device",
		ResourceID: "dev-123",
		UserID:     "user-1",
		Username:   "admin",
		IPAddress:  "192.168.1.1",
		Changes:    `{"name":"test"}`,
		Status:     "success",
		Source:     "api",
	}

	err := store.CreateAuditLog(log)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	if log.ID == "" {
		t.Error("Expected ID to be generated")
	}

	// Get audit log
	retrieved, err := store.GetAuditLog(log.ID)
	if err != nil {
		t.Fatalf("Failed to get audit log: %v", err)
	}

	if retrieved.Action != log.Action {
		t.Errorf("Expected action %s, got %s", log.Action, retrieved.Action)
	}
	if retrieved.Resource != log.Resource {
		t.Errorf("Expected resource %s, got %s", log.Resource, retrieved.Resource)
	}
	if retrieved.Source != log.Source {
		t.Errorf("Expected source %s, got %s", log.Source, retrieved.Source)
	}
}

func TestListAuditLogs(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	// Create multiple audit logs
	logs := []*model.AuditLog{
		{Action: "create", Resource: "device", ResourceID: "dev-1", Status: "success", Source: "api"},
		{Action: "update", Resource: "device", ResourceID: "dev-1", Status: "success", Source: "api"},
		{Action: "delete", Resource: "network", ResourceID: "net-1", Status: "success", Source: "api"},
	}

	for _, log := range logs {
		if err := store.CreateAuditLog(log); err != nil {
			t.Fatalf("Failed to create audit log: %v", err)
		}
	}

	// List all
	all, err := store.ListAuditLogs(&model.AuditFilter{})
	if err != nil {
		t.Fatalf("Failed to list audit logs: %v", err)
	}

	if len(all) < 3 {
		t.Errorf("Expected at least 3 logs, got %d", len(all))
	}

	// Filter by resource
	deviceLogs, err := store.ListAuditLogs(&model.AuditFilter{Resource: "device"})
	if err != nil {
		t.Fatalf("Failed to list device logs: %v", err)
	}

	if len(deviceLogs) != 2 {
		t.Errorf("Expected 2 device logs, got %d", len(deviceLogs))
	}

	// Filter by resource ID
	dev1Logs, err := store.ListAuditLogs(&model.AuditFilter{ResourceID: "dev-1"})
	if err != nil {
		t.Fatalf("Failed to list dev-1 logs: %v", err)
	}

	if len(dev1Logs) != 2 {
		t.Errorf("Expected 2 dev-1 logs, got %d", len(dev1Logs))
	}

	// Filter by action
	createLogs, err := store.ListAuditLogs(&model.AuditFilter{Action: "create"})
	if err != nil {
		t.Fatalf("Failed to list create logs: %v", err)
	}

	if len(createLogs) != 1 {
		t.Errorf("Expected 1 create log, got %d", len(createLogs))
	}

	// Filter by source
	apiLogs, err := store.ListAuditLogs(&model.AuditFilter{Source: "api"})
	if err != nil {
		t.Fatalf("Failed to list api logs: %v", err)
	}

	if len(apiLogs) != 3 {
		t.Errorf("Expected 3 api logs, got %d", len(apiLogs))
	}
}

func TestAuditLogPagination(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	// Create 10 audit logs
	for i := 0; i < 10; i++ {
		log := &model.AuditLog{
			Action:   "create",
			Resource: "device",
			Status:   "success",
		}
		if err := store.CreateAuditLog(log); err != nil {
			t.Fatalf("Failed to create audit log: %v", err)
		}
	}

	// Get first page
	page1, err := store.ListAuditLogs(&model.AuditFilter{Limit: 5, Offset: 0})
	if err != nil {
		t.Fatalf("Failed to get page 1: %v", err)
	}

	if len(page1) != 5 {
		t.Errorf("Expected 5 logs in page 1, got %d", len(page1))
	}

	// Get second page
	page2, err := store.ListAuditLogs(&model.AuditFilter{Limit: 5, Offset: 5})
	if err != nil {
		t.Fatalf("Failed to get page 2: %v", err)
	}

	if len(page2) != 5 {
		t.Errorf("Expected 5 logs in page 2, got %d", len(page2))
	}

	// Ensure pages don't overlap
	if page1[0].ID == page2[0].ID {
		t.Error("Pages should not overlap")
	}
}

func TestDeleteOldAuditLogs(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	// Create old log
	oldLog := &model.AuditLog{
		Timestamp: time.Now().AddDate(0, 0, -100),
		Action:    "create",
		Resource:  "device",
		Status:    "success",
	}
	if err := store.CreateAuditLog(oldLog); err != nil {
		t.Fatalf("Failed to create old log: %v", err)
	}

	// Create recent log
	recentLog := &model.AuditLog{
		Action:   "update",
		Resource: "device",
		Status:   "success",
	}
	if err := store.CreateAuditLog(recentLog); err != nil {
		t.Fatalf("Failed to create recent log: %v", err)
	}

	// Delete logs older than 90 days
	if err := store.DeleteOldAuditLogs(90); err != nil {
		t.Fatalf("Failed to delete old logs: %v", err)
	}

	// Verify old log is gone
	_, err := store.GetAuditLog(oldLog.ID)
	if err != ErrAuditLogNotFound {
		t.Error("Expected old log to be deleted")
	}

	// Verify recent log still exists
	_, err = store.GetAuditLog(recentLog.ID)
	if err != nil {
		t.Error("Expected recent log to still exist")
	}
}

func TestAuditLogTimeFilter(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	// Create logs at different times
	logs := []*model.AuditLog{
		{Timestamp: yesterday, Action: "create", Resource: "device", Status: "success"},
		{Timestamp: now, Action: "update", Resource: "device", Status: "success"},
		{Timestamp: tomorrow, Action: "delete", Resource: "device", Status: "success"},
	}

	for _, log := range logs {
		if err := store.CreateAuditLog(log); err != nil {
			t.Fatalf("Failed to create audit log: %v", err)
		}
	}

	// Filter by start time
	startTime := now.Add(-1 * time.Hour)
	filtered, err := store.ListAuditLogs(&model.AuditFilter{StartTime: &startTime})
	if err != nil {
		t.Fatalf("Failed to filter by start time: %v", err)
	}

	if len(filtered) < 2 {
		t.Errorf("Expected at least 2 logs after start time, got %d", len(filtered))
	}

	// Filter by end time
	endTime := now.Add(1 * time.Hour)
	filtered, err = store.ListAuditLogs(&model.AuditFilter{EndTime: &endTime})
	if err != nil {
		t.Fatalf("Failed to filter by end time: %v", err)
	}

	if len(filtered) < 2 {
		t.Errorf("Expected at least 2 logs before end time, got %d", len(filtered))
	}
}
