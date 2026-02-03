package storage

import (
	"context"
	"fmt"

	"github.com/martinsuchenak/rackd/internal/model"
)

// BulkResult represents the result of a bulk operation
type BulkResult struct {
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// BulkCreateDevices creates multiple devices in a transaction
func (s *SQLiteStorage) BulkCreateDevices(ctx context.Context, devices []*model.Device) (*BulkResult, error) {
	result := &BulkResult{Total: len(devices)}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, device := range devices {
		if err := s.createDeviceInTx(ctx, tx, device); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", device.Name, err))
		} else {
			result.Success++
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
	defer tx.Rollback()

	for _, device := range devices {
		if err := s.updateDeviceInTx(ctx, tx, device); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", device.ID, err))
		} else {
			result.Success++
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
	defer tx.Rollback()

	for _, id := range ids {
		if err := s.deleteDeviceInTx(ctx, tx, id); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", id, err))
		} else {
			result.Success++
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
	defer tx.Rollback()

	for _, id := range deviceIDs {
		// Get existing tags within transaction
		rows, err := tx.QueryContext(ctx, `SELECT tag FROM tags WHERE device_id = ?`, id)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", id, err))
			continue
		}

		existingTags := make(map[string]bool)
		for rows.Next() {
			var tag string
			if err := rows.Scan(&tag); err != nil {
				rows.Close()
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("device %s: %v", id, err))
				continue
			}
			existingTags[tag] = true
		}
		rows.Close()

		// Add new tags
		for _, tag := range tags {
			if !existingTags[tag] {
				_, err := tx.ExecContext(ctx, `INSERT INTO tags (device_id, tag) VALUES (?, ?)`, id, tag)
				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("device %s tag %s: %v", id, tag, err))
					break
				}
			}
		}
		result.Success++
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
	defer tx.Rollback()

	for _, id := range deviceIDs {
		// Delete specified tags
		for _, tag := range tags {
			_, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE device_id = ? AND tag = ?`, id, tag)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("device %s tag %s: %v", id, tag, err))
				break
			}
		}
		result.Success++
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
	defer tx.Rollback()

	for _, network := range networks {
		if err := s.createNetworkInTx(ctx, tx, network); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("network %s: %v", network.Name, err))
		} else {
			result.Success++
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
	defer tx.Rollback()

	for _, id := range ids {
		if err := s.deleteNetworkInTx(ctx, tx, id); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("network %s: %v", id, err))
		} else {
			result.Success++
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditLog(ctx, "bulk_delete", "network", "", map[string]interface{}{"count": len(ids)})
	return result, nil
}
