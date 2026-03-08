package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Conflict operations

// CreateConflict creates a new conflict record
func (s *SQLiteStorage) CreateConflict(ctx context.Context, conflict *model.Conflict) error {
	if conflict == nil {
		return fmt.Errorf("conflict is nil")
	}

	// Generate ID if not provided
	if conflict.ID == "" {
		conflict.ID = newUUID()
	}

	if conflict.DetectedAt.IsZero() {
		conflict.DetectedAt = time.Now().UTC()
	}

	// Convert arrays to JSON for storage
	deviceIDsJSON, err := json.Marshal(conflict.DeviceIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal device IDs: %w", err)
	}
	deviceNamesJSON, err := json.Marshal(conflict.DeviceNames)
	if err != nil {
		return fmt.Errorf("failed to marshal device names: %w", err)
	}
	networkIDsJSON, err := json.Marshal(conflict.NetworkIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal network IDs: %w", err)
	}
	networkNamesJSON, err := json.Marshal(conflict.NetworkNames)
	if err != nil {
		return fmt.Errorf("failed to marshal network names: %w", err)
	}
	subnetsJSON, err := json.Marshal(conflict.Subnets)
	if err != nil {
		return fmt.Errorf("failed to marshal subnets: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO conflicts (
			id, type, status, description, ip_address, device_ids, device_names,
			network_ids, network_names, subnets, detected_at, resolved_at, resolved_by, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, conflict.ID, string(conflict.Type), string(conflict.Status), conflict.Description,
		nullString(conflict.IPAddress), string(deviceIDsJSON), string(deviceNamesJSON),
		string(networkIDsJSON), string(networkNamesJSON), string(subnetsJSON),
		conflict.DetectedAt, nullTime(conflict.ResolvedAt), nullString(conflict.ResolvedBy),
		nullString(conflict.Notes))

	if err != nil {
		return fmt.Errorf("failed to create conflict: %w", err)
	}

	return nil
}

// GetConflict retrieves a conflict by ID
func (s *SQLiteStorage) GetConflict(ctx context.Context, id string) (*model.Conflict, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	var conflict model.Conflict
	var ipAddress, resolvedBy, notes sql.NullString
	var resolvedAt sql.NullTime
	var deviceIDsJSON, deviceNamesJSON, networkIDsJSON, networkNamesJSON, subnetsJSON sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, status, description, ip_address, device_ids, device_names,
		       network_ids, network_names, subnets, detected_at, resolved_at, resolved_by, notes
		FROM conflicts WHERE id = ?
	`, id).Scan(
		&conflict.ID, &conflict.Type, &conflict.Status, &conflict.Description,
		&ipAddress, &deviceIDsJSON, &deviceNamesJSON,
		&networkIDsJSON, &networkNamesJSON, &subnetsJSON,
		&conflict.DetectedAt, &resolvedAt, &resolvedBy, &notes,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conflict not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conflict: %w", err)
	}

	if ipAddress.Valid {
		conflict.IPAddress = ipAddress.String
	}
	if resolvedBy.Valid {
		conflict.ResolvedBy = resolvedBy.String
	}
	if notes.Valid {
		conflict.Notes = notes.String
	}
	if resolvedAt.Valid {
		conflict.ResolvedAt = &resolvedAt.Time
	}

	// Unmarshal JSON arrays
	if deviceIDsJSON.Valid && deviceIDsJSON.String != "" {
		json.Unmarshal([]byte(deviceIDsJSON.String), &conflict.DeviceIDs)
	}
	if deviceNamesJSON.Valid && deviceNamesJSON.String != "" {
		json.Unmarshal([]byte(deviceNamesJSON.String), &conflict.DeviceNames)
	}
	if networkIDsJSON.Valid && networkIDsJSON.String != "" {
		json.Unmarshal([]byte(networkIDsJSON.String), &conflict.NetworkIDs)
	}
	if networkNamesJSON.Valid && networkNamesJSON.String != "" {
		json.Unmarshal([]byte(networkNamesJSON.String), &conflict.NetworkNames)
	}
	if subnetsJSON.Valid && subnetsJSON.String != "" {
		json.Unmarshal([]byte(subnetsJSON.String), &conflict.Subnets)
	}

	return &conflict, nil
}

// ListConflicts retrieves conflicts matching filter criteria
func (s *SQLiteStorage) ListConflicts(ctx context.Context, filter *model.ConflictFilter) ([]model.Conflict, error) {

	query := `SELECT id, type, status, description, ip_address, device_ids, device_names,
		          network_ids, network_names, subnets, detected_at, resolved_at, resolved_by, notes
	          FROM conflicts`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.Type != "" {
			conditions = append(conditions, "type = ?")
			args = append(args, string(filter.Type))
		}
		if filter.Status != "" {
			conditions = append(conditions, "status = ?")
			args = append(args, string(filter.Status))
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY detected_at DESC"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list conflicts: %w", err)
	}
	defer rows.Close()

	var conflicts []model.Conflict
	for rows.Next() {
		var conflict model.Conflict
		var ipAddress, resolvedBy, notes sql.NullString
		var resolvedAt sql.NullTime
		var deviceIDsJSON, deviceNamesJSON, networkIDsJSON, networkNamesJSON, subnetsJSON sql.NullString

		if err := rows.Scan(
			&conflict.ID, &conflict.Type, &conflict.Status, &conflict.Description,
			&ipAddress, &deviceIDsJSON, &deviceNamesJSON,
			&networkIDsJSON, &networkNamesJSON, &subnetsJSON,
			&conflict.DetectedAt, &resolvedAt, &resolvedBy, &notes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan conflict: %w", err)
		}

		if ipAddress.Valid {
			conflict.IPAddress = ipAddress.String
		}
		if resolvedBy.Valid {
			conflict.ResolvedBy = resolvedBy.String
		}
		if notes.Valid {
			conflict.Notes = notes.String
		}
		if resolvedAt.Valid {
			conflict.ResolvedAt = &resolvedAt.Time
		}

		// Unmarshal JSON arrays
		if deviceIDsJSON.Valid && deviceIDsJSON.String != "" {
			json.Unmarshal([]byte(deviceIDsJSON.String), &conflict.DeviceIDs)
		}
		if deviceNamesJSON.Valid && deviceNamesJSON.String != "" {
			json.Unmarshal([]byte(deviceNamesJSON.String), &conflict.DeviceNames)
		}
		if networkIDsJSON.Valid && networkIDsJSON.String != "" {
			json.Unmarshal([]byte(networkIDsJSON.String), &conflict.NetworkIDs)
		}
		if networkNamesJSON.Valid && networkNamesJSON.String != "" {
			json.Unmarshal([]byte(networkNamesJSON.String), &conflict.NetworkNames)
		}
		if subnetsJSON.Valid && subnetsJSON.String != "" {
			json.Unmarshal([]byte(subnetsJSON.String), &conflict.Subnets)
		}

		conflicts = append(conflicts, conflict)
	}

	if conflicts == nil {
		conflicts = []model.Conflict{}
	}

	return conflicts, nil
}

// UpdateConflictStatus updates the status of a conflict
func (s *SQLiteStorage) UpdateConflictStatus(ctx context.Context, id string, status model.ConflictStatus, resolvedBy, notes string) error {
	if id == "" {
		return ErrInvalidID
	}

	var resolvedAt interface{}
	if status == model.ConflictStatusResolved {
		resolvedAt = time.Now().UTC()
	} else {
		resolvedAt = nil
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE conflicts SET status = ?, resolved_at = ?, resolved_by = ?, notes = ?
		WHERE id = ?
	`, string(status), nullTimePtr(resolvedAt), nullString(resolvedBy), nullString(notes), id)

	if err != nil {
		return fmt.Errorf("failed to update conflict status: %w", err)
	}

	return nil
}

// DeleteConflict removes a conflict by ID
func (s *SQLiteStorage) DeleteConflict(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	_, err := s.db.ExecContext(ctx, `DELETE FROM conflicts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete conflict: %w", err)
	}

	return nil
}

// FindDuplicateIPs finds all IP addresses that are assigned to multiple devices
func (s *SQLiteStorage) FindDuplicateIPs(ctx context.Context) ([]model.Conflict, error) {
	// Find IPs that appear more than once in the addresses table
	rows, err := s.db.QueryContext(ctx, `
		SELECT ip, GROUP_CONCAT(device_id) as device_ids, GROUP_CONCAT(d.name) as device_names, COUNT(*) as count
		FROM addresses a
		JOIN devices d ON a.device_id = d.id
		WHERE ip != '' AND ip IS NOT NULL
		GROUP BY ip
		HAVING count > 1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to find duplicate IPs: %w", err)
	}
	defer rows.Close()

	var conflicts []model.Conflict
	for rows.Next() {
		var ip string
		var deviceIDsStr, deviceNamesStr string
		var count int

		if err := rows.Scan(&ip, &deviceIDsStr, &deviceNamesStr, &count); err != nil {
			return nil, fmt.Errorf("failed to scan duplicate IP: %w", err)
		}

		deviceIDs := strings.Split(deviceIDsStr, ",")
		deviceNames := strings.Split(deviceNamesStr, ",")

		conflicts = append(conflicts, model.Conflict{
			ID:          newUUID(),
			Type:        model.ConflictTypeDuplicateIP,
			Status:      model.ConflictStatusActive,
			Description: fmt.Sprintf("IP address %s is assigned to %d devices", ip, count),
			IPAddress:   ip,
			DeviceIDs:   deviceIDs,
			DeviceNames: deviceNames,
			DetectedAt:  time.Now().UTC(),
		})
	}

	if conflicts == nil {
		conflicts = []model.Conflict{}
	}

	return conflicts, nil
}

// FindOverlappingSubnets finds all network subnets that overlap
func (s *SQLiteStorage) FindOverlappingSubnets(ctx context.Context) ([]model.Conflict, error) {
	// Get all networks
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, subnet FROM networks ORDER BY subnet
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}
	defer rows.Close()

	type networkInfo struct {
		ID     string
		Name   string
		Subnet string
		IPNet  *net.IPNet
	}

	var networks []networkInfo
	for rows.Next() {
		var ni networkInfo
		if err := rows.Scan(&ni.ID, &ni.Name, &ni.Subnet); err != nil {
			return nil, fmt.Errorf("failed to scan network: %w", err)
		}

		// Parse the subnet
		_, ipNet, err := net.ParseCIDR(ni.Subnet)
		if err != nil {
			// Skip invalid subnets
			continue
		}
		ni.IPNet = ipNet
		networks = append(networks, ni)
	}

	// Find overlaps
	var conflicts []model.Conflict
	checked := make(map[string]bool)

	for i, n1 := range networks {
		for j := i + 1; j < len(networks); j++ {
			n2 := networks[j]

			// Create a unique key for this pair to avoid duplicates
			key1 := n1.ID + "-" + n2.ID
			key2 := n2.ID + "-" + n1.ID
			if checked[key1] || checked[key2] {
				continue
			}
			checked[key1] = true
			checked[key2] = true

			// Check for overlap
			if networksOverlap(n1.IPNet, n2.IPNet) {
				conflicts = append(conflicts, model.Conflict{
					ID:           newUUID(),
					Type:         model.ConflictTypeOverlappingSubnet,
					Status:       model.ConflictStatusActive,
					Description:  fmt.Sprintf("Subnets %s and %s overlap", n1.Subnet, n2.Subnet),
					NetworkIDs:   []string{n1.ID, n2.ID},
					NetworkNames: []string{n1.Name, n2.Name},
					Subnets:      []string{n1.Subnet, n2.Subnet},
					DetectedAt:   time.Now().UTC(),
				})
			}
		}
	}

	if conflicts == nil {
		conflicts = []model.Conflict{}
	}

	return conflicts, nil
}

// networksOverlap checks if two IP networks overlap
func networksOverlap(n1, n2 *net.IPNet) bool {
	return n1.Contains(n2.IP) || n2.Contains(n1.IP)
}

// GetConflictsByDeviceID returns all conflicts involving a specific device
func (s *SQLiteStorage) GetConflictsByDeviceID(ctx context.Context, deviceID string) ([]model.Conflict, error) {
	if deviceID == "" {
		return nil, ErrInvalidID
	}

	// Use JSON extraction to find conflicts containing this device ID
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, status, description, ip_address, device_ids, device_names,
		       network_ids, network_names, subnets, detected_at, resolved_at, resolved_by, notes
		FROM conflicts
		WHERE type = 'duplicate_ip'
		AND (device_ids LIKE '%' || ? || '%' OR device_ids LIKE '%,' || ? || '%')
		ORDER BY detected_at DESC
	`, deviceID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conflicts by device: %w", err)
	}
	defer rows.Close()

	return scanConflicts(ctx, rows)
}

// GetConflictsByIP returns all conflicts for a specific IP address
func (s *SQLiteStorage) GetConflictsByIP(ctx context.Context, ip string) ([]model.Conflict, error) {
	if ip == "" {
		return nil, ErrInvalidID
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, status, description, ip_address, device_ids, device_names,
		       network_ids, network_names, subnets, detected_at, resolved_at, resolved_by, notes
		FROM conflicts
		WHERE ip_address = ?
		ORDER BY detected_at DESC
	`, ip)
	if err != nil {
		return nil, fmt.Errorf("failed to get conflicts by IP: %w", err)
	}
	defer rows.Close()

	return scanConflicts(ctx, rows)
}

// MarkConflictsResolvedForDevice marks all active conflicts for a device as resolved
func (s *SQLiteStorage) MarkConflictsResolvedForDevice(ctx context.Context, deviceID, resolvedBy string) error {
	// Get conflicts involving this device
	conflicts, err := s.GetConflictsByDeviceID(ctx, deviceID)
	if err != nil {
		return err
	}

	// Mark them as resolved
	now := time.Now().UTC()
	for _, c := range conflicts {
		if c.Status == model.ConflictStatusActive {
			_, err := s.db.ExecContext(ctx, `
				UPDATE conflicts SET status = ?, resolved_at = ?, resolved_by = ?
				WHERE id = ?
			`, string(model.ConflictStatusResolved), now, resolvedBy, c.ID)
			if err != nil {
				return fmt.Errorf("failed to mark conflict resolved: %w", err)
			}
		}
	}

	return nil
}

// scanConflicts is a helper to scan conflict rows
func scanConflicts(ctx context.Context, rows *sql.Rows) ([]model.Conflict, error) {
	var conflicts []model.Conflict
	for rows.Next() {
		var conflict model.Conflict
		var ipAddress, resolvedBy, notes sql.NullString
		var resolvedAt sql.NullTime
		var deviceIDsJSON, deviceNamesJSON, networkIDsJSON, networkNamesJSON, subnetsJSON sql.NullString

		if err := rows.Scan(
			&conflict.ID, &conflict.Type, &conflict.Status, &conflict.Description,
			&ipAddress, &deviceIDsJSON, &deviceNamesJSON,
			&networkIDsJSON, &networkNamesJSON, &subnetsJSON,
			&conflict.DetectedAt, &resolvedAt, &resolvedBy, &notes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan conflict: %w", err)
		}

		if ipAddress.Valid {
			conflict.IPAddress = ipAddress.String
		}
		if resolvedBy.Valid {
			conflict.ResolvedBy = resolvedBy.String
		}
		if notes.Valid {
			conflict.Notes = notes.String
		}
		if resolvedAt.Valid {
			conflict.ResolvedAt = &resolvedAt.Time
		}

		// Unmarshal JSON arrays
		if deviceIDsJSON.Valid && deviceIDsJSON.String != "" {
			json.Unmarshal([]byte(deviceIDsJSON.String), &conflict.DeviceIDs)
		}
		if deviceNamesJSON.Valid && deviceNamesJSON.String != "" {
			json.Unmarshal([]byte(deviceNamesJSON.String), &conflict.DeviceNames)
		}
		if networkIDsJSON.Valid && networkIDsJSON.String != "" {
			json.Unmarshal([]byte(networkIDsJSON.String), &conflict.NetworkIDs)
		}
		if networkNamesJSON.Valid && networkNamesJSON.String != "" {
			json.Unmarshal([]byte(networkNamesJSON.String), &conflict.NetworkNames)
		}
		if subnetsJSON.Valid && subnetsJSON.String != "" {
			json.Unmarshal([]byte(subnetsJSON.String), &conflict.Subnets)
		}

		conflicts = append(conflicts, conflict)
	}

	if conflicts == nil {
		conflicts = []model.Conflict{}
	}

	return conflicts, nil
}

// nullTime returns a sql.NullTime for nil time pointers
func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// nullTimePtr returns a sql.NullTime for an interface{} that might be a time.Time or nil
func nullTimePtr(t interface{}) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	if tm, ok := t.(time.Time); ok {
		return sql.NullTime{Time: tm, Valid: true}
	}
	return sql.NullTime{}
}
