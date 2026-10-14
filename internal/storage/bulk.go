package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// BulkResult represents the result of a bulk operation
type BulkResult struct {
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
	// SucceededIDs are the IDs of the items that succeeded, for publishing
	// events. They are not part of the API response.
	SucceededIDs []string `json:"-"`
}

// BulkCreateDevices creates multiple devices in a transaction
func (s *SQLiteStorage) BulkCreateDevices(ctx context.Context, devices []*model.Device) (*BulkResult, error) {
	result := &BulkResult{Total: len(devices)}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, device := range devices {
		if err := s.createDeviceInTx(ctx, tx, device); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", device.Name, err))
		} else {
			result.Success++
			result.SucceededIDs = append(result.SucceededIDs, device.ID)
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditLog(ctx, "bulk_create", "device", "", map[string]interface{}{"count": len(devices)})
	return result, nil
}

// BulkUpdateDevices updates multiple devices in a transaction
func (s *SQLiteStorage) BulkUpdateDevices(ctx context.Context, devices []*model.Device) (*BulkResult, error) {
	result := &BulkResult{Total: len(devices)}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, device := range devices {
		if err := s.updateDeviceInTx(ctx, tx, device); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", device.ID, err))
		} else {
			result.Success++
			result.SucceededIDs = append(result.SucceededIDs, device.ID)
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditLog(ctx, "bulk_update", "device", "", map[string]interface{}{"count": len(devices)})
	return result, nil
}

// BulkDeleteDevices deletes multiple devices in a transaction
func (s *SQLiteStorage) BulkDeleteDevices(ctx context.Context, ids []string) (*BulkResult, error) {
	result := &BulkResult{Total: len(ids)}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range ids {
		if err := s.deleteDeviceInTx(ctx, tx, id); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", id, err))
		} else {
			result.Success++
			result.SucceededIDs = append(result.SucceededIDs, id)
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditLog(ctx, "bulk_delete", "device", "", map[string]interface{}{"count": len(ids)})
	return result, nil
}

// BulkAddTags adds tags to multiple devices
func (s *SQLiteStorage) BulkAddTags(ctx context.Context, deviceIDs []string, tags []string) (*BulkResult, error) {
	result := &BulkResult{Total: len(deviceIDs)}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range deviceIDs {
		if err := addTagsInTx(ctx, tx, id, tags); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", id, err))
			continue
		}
		result.Success++
		result.SucceededIDs = append(result.SucceededIDs, id)
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditLog(ctx, "bulk_add_tags", "device", "", map[string]interface{}{"count": len(deviceIDs)})
	return result, nil
}

// BulkRemoveTags removes tags from multiple devices
func (s *SQLiteStorage) BulkRemoveTags(ctx context.Context, deviceIDs []string, tags []string) (*BulkResult, error) {
	result := &BulkResult{Total: len(deviceIDs)}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range deviceIDs {
		if err := removeTagsInTx(ctx, tx, id, tags); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", id, err))
			continue
		}
		result.Success++
		result.SucceededIDs = append(result.SucceededIDs, id)
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditLog(ctx, "bulk_remove_tags", "device", "", map[string]interface{}{"count": len(deviceIDs)})
	return result, nil
}

// BulkCreateNetworks creates multiple networks in a transaction
func (s *SQLiteStorage) BulkCreateNetworks(ctx context.Context, networks []*model.Network) (*BulkResult, error) {
	result := &BulkResult{Total: len(networks)}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, network := range networks {
		if err := s.createNetworkInTx(ctx, tx, network); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("network %s: %v", network.Name, err))
		} else {
			result.Success++
			result.SucceededIDs = append(result.SucceededIDs, network.ID)
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditLog(ctx, "bulk_create", "network", "", map[string]interface{}{"count": len(networks)})
	return result, nil
}

// BulkDeleteNetworks deletes multiple networks in a transaction
func (s *SQLiteStorage) BulkDeleteNetworks(ctx context.Context, ids []string) (*BulkResult, error) {
	result := &BulkResult{Total: len(ids)}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range ids {
		if err := s.deleteNetworkInTx(ctx, tx, id); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("network %s: %v", id, err))
		} else {
			result.Success++
			result.SucceededIDs = append(result.SucceededIDs, id)
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditLog(ctx, "bulk_delete", "network", "", map[string]interface{}{"count": len(ids)})
	return result, nil
}

// addTagsInTx adds the tags a device doesn't have yet.
func addTagsInTx(ctx context.Context, tx *sql.Tx, deviceID string, tags []string) error {
	rows, err := tx.QueryContext(ctx, `SELECT tag FROM tags WHERE device_id = ?`, deviceID)
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			_ = rows.Close()
			return err
		}
		existing[tag] = true
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, tag := range tags {
		if existing[tag] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags (device_id, tag) VALUES (?, ?)`, deviceID, tag); err != nil {
			return fmt.Errorf("tag %s: %w", tag, err)
		}
	}
	return nil
}

// removeTagsInTx removes tags from a device.
func removeTagsInTx(ctx context.Context, tx *sql.Tx, deviceID string, tags []string) error {
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE device_id = ? AND tag = ?`, deviceID, tag); err != nil {
			return fmt.Errorf("tag %s: %w", tag, err)
		}
	}
	return nil
}
