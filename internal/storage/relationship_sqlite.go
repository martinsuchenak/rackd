package storage

import (
	"context"
	"database/sql"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Relationship operations

func (s *SQLiteStorage) AddRelationship(ctx context.Context, parentID, childID, relationshipType, notes string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO device_relationships (parent_id, child_id, type, notes)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (parent_id, child_id, type) DO UPDATE SET notes = excluded.notes
	`, parentID, childID, relationshipType, notes)
	if err != nil {
		return err
	}
	s.auditLog(ctx, "add", "relationship", parentID+":"+childID, nil)
	return nil
}

func (s *SQLiteStorage) RemoveRelationship(ctx context.Context, parentID, childID, relationshipType string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM device_relationships
		WHERE parent_id = ? AND child_id = ? AND type = ?
	`, parentID, childID, relationshipType)
	if err != nil {
		return err
	}
	s.auditLog(ctx, "remove", "relationship", parentID+":"+childID, nil)
	return nil
}

func (s *SQLiteStorage) UpdateRelationshipNotes(ctx context.Context, parentID, childID, relationshipType, notes string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE device_relationships
		SET notes = ?
		WHERE parent_id = ? AND child_id = ? AND type = ?
	`, notes, parentID, childID, relationshipType)
	if err != nil {
		return err
	}
	s.auditLog(ctx, "update", "relationship", parentID+":"+childID, nil)
	return nil
}

func (s *SQLiteStorage) GetRelationships(ctx context.Context, deviceID string) ([]model.DeviceRelationship, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT parent_id, child_id, type, notes, created_at
		FROM device_relationships
		WHERE parent_id = ? OR child_id = ?
	`, deviceID, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []model.DeviceRelationship
	for rows.Next() {
		var r model.DeviceRelationship
		if err := rows.Scan(&r.ParentID, &r.ChildID, &r.Type, &r.Notes, &r.CreatedAt); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

func (s *SQLiteStorage) ListAllRelationships(ctx context.Context) ([]model.DeviceRelationship, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT parent_id, child_id, type, notes, created_at
		FROM device_relationships
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []model.DeviceRelationship
	for rows.Next() {
		var r model.DeviceRelationship
		if err := rows.Scan(&r.ParentID, &r.ChildID, &r.Type, &r.Notes, &r.CreatedAt); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

func (s *SQLiteStorage) GetRelatedDevices(ctx context.Context, deviceID, relationshipType string) ([]model.Device, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.name, d.description, d.make_model, d.os, d.datacenter_id,
		       d.username, d.location, d.created_at, d.updated_at
		FROM devices d
		JOIN device_relationships r ON (d.id = r.child_id OR d.id = r.parent_id)
		WHERE (r.parent_id = ? OR r.child_id = ?) AND r.type = ? AND d.id != ?
	`, deviceID, deviceID, relationshipType, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var d model.Device
		var dcID sql.NullString
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.MakeModel, &d.OS,
			&dcID, &d.Username, &d.Location, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.DatacenterID = dcID.String
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch related data after closing rows
	for i := range devices {
		devices[i].Addresses, _ = s.getDeviceAddresses(ctx, devices[i].ID)
		devices[i].Tags, _ = s.getDeviceTags(ctx, devices[i].ID)
		devices[i].Domains, _ = s.getDeviceDomains(ctx, devices[i].ID)
	}
	return devices, nil
}
