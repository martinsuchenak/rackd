package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
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
		conflict.DetectedAt = nowUTC()
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
		if field, ok := DuplicateField(err); ok && field == "ip_address" {
			return ErrConflictRecorded
		}
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
		return nil, ErrConflictNotFound
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
		_ = json.Unmarshal([]byte(deviceIDsJSON.String), &conflict.DeviceIDs)
	}
	if deviceNamesJSON.Valid && deviceNamesJSON.String != "" {
		_ = json.Unmarshal([]byte(deviceNamesJSON.String), &conflict.DeviceNames)
	}
	if networkIDsJSON.Valid && networkIDsJSON.String != "" {
		_ = json.Unmarshal([]byte(networkIDsJSON.String), &conflict.NetworkIDs)
	}
	if networkNamesJSON.Valid && networkNamesJSON.String != "" {
		_ = json.Unmarshal([]byte(networkNamesJSON.String), &conflict.NetworkNames)
	}
	if subnetsJSON.Valid && subnetsJSON.String != "" {
		_ = json.Unmarshal([]byte(subnetsJSON.String), &conflict.Subnets)
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
	defer func() { _ = rows.Close() }()

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
			_ = json.Unmarshal([]byte(deviceIDsJSON.String), &conflict.DeviceIDs)
		}
		if deviceNamesJSON.Valid && deviceNamesJSON.String != "" {
			_ = json.Unmarshal([]byte(deviceNamesJSON.String), &conflict.DeviceNames)
		}
		if networkIDsJSON.Valid && networkIDsJSON.String != "" {
			_ = json.Unmarshal([]byte(networkIDsJSON.String), &conflict.NetworkIDs)
		}
		if networkNamesJSON.Valid && networkNamesJSON.String != "" {
			_ = json.Unmarshal([]byte(networkNamesJSON.String), &conflict.NetworkNames)
		}
		if subnetsJSON.Valid && subnetsJSON.String != "" {
			_ = json.Unmarshal([]byte(subnetsJSON.String), &conflict.Subnets)
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
		resolvedAt = nowUTC()
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

	res, err := s.db.ExecContext(ctx, `DELETE FROM conflicts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete conflict: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return ErrConflictNotFound
	}

	return nil
}

// FindDuplicateIPs finds all IP addresses that are assigned to multiple devices
func (s *SQLiteStorage) FindDuplicateIPs(ctx context.Context) ([]model.Conflict, error) {
	// IPs held by more than one device. A device listing an IP twice isn't a
	// duplicate, so devices are counted, not addresses, and names are read as
	// rows rather than split from a list (a name can contain a comma).
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT a.ip, d.id, d.name
		FROM addresses a
		JOIN devices d ON a.device_id = d.id
		WHERE a.ip IN (
			SELECT ip FROM addresses
			WHERE ip != '' AND ip IS NOT NULL
			GROUP BY ip
			HAVING COUNT(DISTINCT device_id) > 1
		)
		ORDER BY a.ip, d.name, d.id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to find duplicate IPs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	conflicts := []model.Conflict{}
	for rows.Next() {
		var ip, deviceID, deviceName string
		if err := rows.Scan(&ip, &deviceID, &deviceName); err != nil {
			return nil, fmt.Errorf("failed to scan duplicate IP: %w", err)
		}
		if n := len(conflicts); n == 0 || conflicts[n-1].IPAddress != ip {
			conflicts = append(conflicts, model.Conflict{
				ID:         newUUID(),
				Type:       model.ConflictTypeDuplicateIP,
				Status:     model.ConflictStatusActive,
				IPAddress:  ip,
				DetectedAt: nowUTC(),
			})
		}
		c := &conflicts[len(conflicts)-1]
		c.DeviceIDs = append(c.DeviceIDs, deviceID)
		c.DeviceNames = append(c.DeviceNames, deviceName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range conflicts {
		conflicts[i].Description = DuplicateIPDescription(conflicts[i].IPAddress, len(conflicts[i].DeviceIDs))
	}
	return conflicts, nil
}

// DuplicateIPDescription describes a duplicate-IP conflict.
func DuplicateIPDescription(ip string, devices int) string {
	return fmt.Sprintf("IP address %s is assigned to %d devices", ip, devices)
}

// ResolveOpenConflict resolves a conflict that isn't resolved yet, and
// reports whether it did, so concurrent callers resolve it once.
func (s *SQLiteStorage) ResolveOpenConflict(ctx context.Context, id, resolvedBy, notes string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE conflicts SET status = ?, resolved_at = ?, resolved_by = ?, notes = ?
		WHERE id = ? AND status != ?
	`, string(model.ConflictStatusResolved), nowUTC(), nullString(resolvedBy), nullString(notes), id, string(model.ConflictStatusResolved))
	if err != nil {
		return false, fmt.Errorf("failed to resolve conflict: %w", err)
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// UpdateConflictParties records the devices or networks a conflict now
// involves, with its description, leaving its status and notes alone.
func (s *SQLiteStorage) UpdateConflictParties(ctx context.Context, conflict *model.Conflict) error {
	if conflict.ID == "" {
		return ErrInvalidID
	}
	fields := map[string]any{}
	for name, v := range map[string]any{"device_ids": conflict.DeviceIDs, "device_names": conflict.DeviceNames,
		"network_ids": conflict.NetworkIDs, "network_names": conflict.NetworkNames, "subnets": conflict.Subnets} {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %w", name, err)
		}
		fields[name] = string(b)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE conflicts SET description = ?, device_ids = ?, device_names = ?,
		       network_ids = ?, network_names = ?, subnets = ?
		WHERE id = ?
	`, conflict.Description, fields["device_ids"], fields["device_names"],
		fields["network_ids"], fields["network_names"], fields["subnets"], conflict.ID)
	if err != nil {
		return fmt.Errorf("failed to update conflict: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrConflictNotFound
	}
	return nil
}

// MaxOverlapConflicts caps how many overlapping pairs one detection run
// reports. Networks may overlap freely, so n identical subnets make n²/2 pairs;
// without a cap, a few thousand networks could exhaust memory.
const MaxOverlapConflicts = 1000

// FindOverlappingSubnets finds network subnets that overlap. Pairs known
// reports are skipped and don't count towards the MaxOverlapConflicts cap, so
// repeated runs record every overlap eventually.
func (s *SQLiteStorage) FindOverlappingSubnets(ctx context.Context, known func(networkIDs []string) bool) ([]model.Conflict, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, subnet FROM networks`)
	if err != nil {
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type networkInfo struct {
		ID, Name, Subnet string
		first, last      netip.Addr
	}

	var networks []networkInfo
	for rows.Next() {
		var ni networkInfo
		if err := rows.Scan(&ni.ID, &ni.Name, &ni.Subnet); err != nil {
			return nil, fmt.Errorf("failed to scan network: %w", err)
		}
		prefix, err := netip.ParsePrefix(ni.Subnet)
		if err != nil {
			continue // skip invalid subnets
		}
		prefix = prefix.Masked()
		ni.first, ni.last = prefix.Addr(), lastAddr(prefix)
		networks = append(networks, ni)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read networks: %w", err)
	}

	// Sorted by first address, a network can only overlap the ones after it
	// that start before it ends, so the inner loop stops at the first that
	// doesn't. IPv4 sorts before IPv6, so the families never mix.
	sort.Slice(networks, func(i, j int) bool { return networks[i].first.Less(networks[j].first) })

	conflicts := []model.Conflict{}
	for i, n1 := range networks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, n2 := range networks[i+1:] {
			if n2.first.BitLen() != n1.first.BitLen() || n1.last.Less(n2.first) {
				break
			}
			ids := []string{n1.ID, n2.ID}
			if known != nil && known(ids) {
				continue
			}
			conflicts = append(conflicts, model.Conflict{
				ID:           newUUID(),
				Type:         model.ConflictTypeOverlappingSubnet,
				Status:       model.ConflictStatusActive,
				Description:  fmt.Sprintf("Subnets %s and %s overlap", n1.Subnet, n2.Subnet),
				NetworkIDs:   ids,
				NetworkNames: []string{n1.Name, n2.Name},
				Subnets:      []string{n1.Subnet, n2.Subnet},
				DetectedAt:   nowUTC(),
			})
			if len(conflicts) == MaxOverlapConflicts {
				return conflicts, nil
			}
		}
	}
	return conflicts, nil
}

// lastAddr returns the last address in a masked prefix.
func lastAddr(p netip.Prefix) netip.Addr {
	b := p.Addr().AsSlice()
	for bit := p.Bits(); bit < len(b)*8; bit++ {
		b[bit/8] |= 0x80 >> (bit % 8)
	}
	addr, _ := netip.AddrFromSlice(b)
	return addr
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
	defer func() { _ = rows.Close() }()

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
	defer func() { _ = rows.Close() }()

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
	now := nowUTC()
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
			_ = json.Unmarshal([]byte(deviceIDsJSON.String), &conflict.DeviceIDs)
		}
		if deviceNamesJSON.Valid && deviceNamesJSON.String != "" {
			_ = json.Unmarshal([]byte(deviceNamesJSON.String), &conflict.DeviceNames)
		}
		if networkIDsJSON.Valid && networkIDsJSON.String != "" {
			_ = json.Unmarshal([]byte(networkIDsJSON.String), &conflict.NetworkIDs)
		}
		if networkNamesJSON.Valid && networkNamesJSON.String != "" {
			_ = json.Unmarshal([]byte(networkNamesJSON.String), &conflict.NetworkNames)
		}
		if subnetsJSON.Valid && subnetsJSON.String != "" {
			_ = json.Unmarshal([]byte(subnetsJSON.String), &conflict.Subnets)
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

// DeviceRef identifies a device by ID and name.
type DeviceRef struct {
	ID   string
	Name string
}

// DevicesWithIP returns the devices that have ip among their addresses, by
// name, with one indexed lookup.
func (s *SQLiteStorage) DevicesWithIP(ctx context.Context, ip string) ([]DeviceRef, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT d.id, d.name
		FROM addresses a
		JOIN devices d ON d.id = a.device_id
		WHERE a.ip = ?
		ORDER BY d.name, d.id`, ip)
	if err != nil {
		return nil, fmt.Errorf("failed to find devices with IP %s: %w", ip, err)
	}
	defer func() { _ = rows.Close() }()
	var out []DeviceRef
	for rows.Next() {
		var d DeviceRef
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
