package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Pool operations

// CreateNetworkPool creates a new network pool
func (s *SQLiteStorage) CreateNetworkPool(ctx context.Context, pool *model.NetworkPool) error {
	if pool == nil {
		return fmt.Errorf("pool is nil")
	}

	// Validate network exists
	var networkExists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM networks WHERE id = ?)`, pool.NetworkID).Scan(&networkExists)
	if err != nil {
		return fmt.Errorf("failed to check network existence: %w", err)
	}
	if !networkExists {
		return ErrNetworkNotFound
	}

	// Generate ID if not provided
	if pool.ID == "" {
		pool.ID = newUUID()
	}

	now := time.Now().UTC()
	pool.CreatedAt = now
	pool.UpdatedAt = now

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert pool
	_, err = tx.ExecContext(ctx, `
		INSERT INTO network_pools (id, network_id, name, start_ip, end_ip, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, pool.ID, pool.NetworkID, pool.Name, pool.StartIP, pool.EndIP, pool.Description, pool.CreatedAt, pool.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create network pool: %w", err)
	}

	// Insert tags
	if err := s.insertPoolTags(ctx, tx, pool.ID, pool.Tags); err != nil {
		return fmt.Errorf("failed to insert pool tags: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "create", "pool", pool.ID, pool)
	return nil
}

// insertPoolTags inserts tags for a pool within a transaction
func (s *SQLiteStorage) insertPoolTags(ctx context.Context, tx *sql.Tx, poolID string, tags []string) error {
	for _, tag := range tags {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO pool_tags (pool_id, tag) VALUES (?, ?)
		`, poolID, tag)
		if err != nil {
			return err
		}
	}
	return nil
}

// getPoolTags retrieves all tags for a pool
func (s *SQLiteStorage) getPoolTags(ctx context.Context, poolID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tag FROM pool_tags WHERE pool_id = ?`, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	if tags == nil {
		tags = []string{}
	}

	return tags, rows.Err()
}

// GetNetworkPool retrieves a network pool by ID
func (s *SQLiteStorage) GetNetworkPool(id string) (*model.NetworkPool, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	pool := &model.NetworkPool{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, network_id, name, start_ip, end_ip, description, created_at, updated_at
		FROM network_pools WHERE id = ?
	`, id).Scan(
		&pool.ID, &pool.NetworkID, &pool.Name, &pool.StartIP, &pool.EndIP,
		&pool.Description, &pool.CreatedAt, &pool.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrPoolNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get network pool: %w", err)
	}

	// Get tags
	tags, err := s.getPoolTags(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool tags: %w", err)
	}
	pool.Tags = tags

	return pool, nil
}

// UpdateNetworkPool updates an existing network pool
func (s *SQLiteStorage) UpdateNetworkPool(ctx context.Context, pool *model.NetworkPool) error {
	if pool == nil {
		return fmt.Errorf("pool is nil")
	}
	if pool.ID == "" {
		return ErrInvalidID
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if pool exists
	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM network_pools WHERE id = ?)`, pool.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check pool existence: %w", err)
	}
	if !exists {
		return ErrPoolNotFound
	}

	pool.UpdatedAt = time.Now().UTC()

	// Update pool
	_, err = tx.ExecContext(ctx, `
		UPDATE network_pools SET name = ?, start_ip = ?, end_ip = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, pool.Name, pool.StartIP, pool.EndIP, pool.Description, pool.UpdatedAt, pool.ID)
	if err != nil {
		return fmt.Errorf("failed to update network pool: %w", err)
	}

	// Delete existing tags and reinsert
	if _, err := tx.ExecContext(ctx, `DELETE FROM pool_tags WHERE pool_id = ?`, pool.ID); err != nil {
		return fmt.Errorf("failed to delete pool tags: %w", err)
	}
	if err := s.insertPoolTags(ctx, tx, pool.ID, pool.Tags); err != nil {
		return fmt.Errorf("failed to insert pool tags: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "update", "pool", pool.ID, pool)
	return nil
}

// DeleteNetworkPool removes a network pool by ID
func (s *SQLiteStorage) DeleteNetworkPool(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if pool exists
	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM network_pools WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check pool existence: %w", err)
	}
	if !exists {
		return ErrPoolNotFound
	}

	// Unlink addresses from this pool (set pool_id to NULL)
	_, err = tx.ExecContext(ctx, `UPDATE addresses SET pool_id = NULL WHERE pool_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to unlink addresses: %w", err)
	}

	// Delete pool (tags cascade via foreign key)
	_, err = tx.ExecContext(ctx, `DELETE FROM network_pools WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete network pool: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "delete", "pool", id, nil)
	return nil
}

// ListNetworkPools retrieves pools matching the filter criteria
func (s *SQLiteStorage) ListNetworkPools(filter *model.NetworkPoolFilter) ([]model.NetworkPool, error) {
	ctx := context.Background()

	query := `SELECT id, network_id, name, start_ip, end_ip, description, created_at, updated_at FROM network_pools`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.NetworkID != "" {
			conditions = append(conditions, "network_id = ?")
			args = append(args, filter.NetworkID)
		}

		if len(filter.Tags) > 0 {
			// Match pools that have ALL specified tags
			for _, tag := range filter.Tags {
				conditions = append(conditions, "id IN (SELECT pool_id FROM pool_tags WHERE tag = ?)")
				args = append(args, tag)
			}
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list network pools: %w", err)
	}
	defer rows.Close()

	var pools []model.NetworkPool
	for rows.Next() {
		var pool model.NetworkPool
		if err := rows.Scan(
			&pool.ID, &pool.NetworkID, &pool.Name, &pool.StartIP, &pool.EndIP,
			&pool.Description, &pool.CreatedAt, &pool.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan network pool: %w", err)
		}
		pools = append(pools, pool)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load tags for each pool
	for i := range pools {
		tags, err := s.getPoolTags(ctx, pools[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for pool %s: %w", pools[i].ID, err)
		}
		pools[i].Tags = tags
	}

	if pools == nil {
		pools = []model.NetworkPool{}
	}

	return pools, nil
}

// GetNextAvailableIP finds the first unused IP address in a pool's range
func (s *SQLiteStorage) GetNextAvailableIP(poolID string) (string, error) {
	if poolID == "" {
		return "", ErrInvalidID
	}

	ctx := context.Background()

	// Get the pool
	pool, err := s.GetNetworkPool(poolID)
	if err != nil {
		return "", err
	}

	// Parse start and end IPs
	startIP := net.ParseIP(pool.StartIP)
	endIP := net.ParseIP(pool.EndIP)

	if startIP == nil || endIP == nil {
		return "", fmt.Errorf("invalid IP range: %s - %s", pool.StartIP, pool.EndIP)
	}

	// Convert to IPv4 if applicable
	startIP = startIP.To4()
	endIP = endIP.To4()

	if startIP == nil || endIP == nil {
		// Handle IPv6 or other formats
		return "", fmt.Errorf("only IPv4 addresses are currently supported")
	}

	// Get all used IPs in this pool
	usedIPs := make(map[string]bool)
	rows, err := s.db.QueryContext(ctx, `SELECT ip FROM addresses WHERE pool_id = ?`, poolID)
	if err != nil {
		return "", fmt.Errorf("failed to query used IPs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return "", fmt.Errorf("failed to scan IP: %w", err)
		}
		usedIPs[ip] = true
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	// Iterate through the range to find the first available IP
	current := make(net.IP, len(startIP))
	copy(current, startIP)

	for {
		ipStr := current.String()
		if !usedIPs[ipStr] {
			return ipStr, nil
		}

		// Increment IP
		if !incrementIP(current, endIP) {
			break
		}
	}

	return "", ErrIPNotAvailable
}

// incrementIP increments an IP address by 1, returns false if it exceeds endIP
func incrementIP(ip net.IP, endIP net.IP) bool {
	// Increment from the last byte
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}

	// Check if we've exceeded the end IP
	for i := range ip {
		if ip[i] < endIP[i] {
			return true
		}
		if ip[i] > endIP[i] {
			return false
		}
	}

	// IPs are equal, this is the last valid IP
	return true
}

// ValidateIPInPool checks if an IP address is within a pool's range
func (s *SQLiteStorage) ValidateIPInPool(poolID, ip string) (bool, error) {
	if poolID == "" {
		return false, ErrInvalidID
	}

	// Get the pool
	pool, err := s.GetNetworkPool(poolID)
	if err != nil {
		return false, err
	}

	// Parse all IPs
	checkIP := net.ParseIP(ip)
	startIP := net.ParseIP(pool.StartIP)
	endIP := net.ParseIP(pool.EndIP)

	if checkIP == nil || startIP == nil || endIP == nil {
		return false, fmt.Errorf("invalid IP address")
	}

	// Convert to IPv4 if applicable
	checkIP = checkIP.To4()
	startIP = startIP.To4()
	endIP = endIP.To4()

	if checkIP == nil || startIP == nil || endIP == nil {
		return false, fmt.Errorf("only IPv4 addresses are currently supported")
	}

	// Check if checkIP is within range [startIP, endIP]
	return ipInRange(checkIP, startIP, endIP), nil
}

// ipInRange checks if ip is within the range [start, end] (inclusive)
func ipInRange(ip, start, end net.IP) bool {
	// Compare with start: ip >= start
	for i := range ip {
		if ip[i] < start[i] {
			return false
		}
		if ip[i] > start[i] {
			break
		}
	}

	// Compare with end: ip <= end
	for i := range ip {
		if ip[i] > end[i] {
			return false
		}
		if ip[i] < end[i] {
			break
		}
	}

	return true
}

// GetPoolHeatmap returns the status of all IPs in a pool's range
func (s *SQLiteStorage) GetPoolHeatmap(poolID string) ([]IPStatus, error) {
	if poolID == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	// Get the pool
	pool, err := s.GetNetworkPool(poolID)
	if err != nil {
		return nil, err
	}

	// Parse start and end IPs
	startIP := net.ParseIP(pool.StartIP)
	endIP := net.ParseIP(pool.EndIP)

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("invalid IP range: %s - %s", pool.StartIP, pool.EndIP)
	}

	// Convert to IPv4
	startIP = startIP.To4()
	endIP = endIP.To4()

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("only IPv4 addresses are currently supported")
	}

	// Get all addresses in this pool with their device IDs
	addressMap := make(map[string]string) // ip -> device_id
	rows, err := s.db.QueryContext(ctx, `SELECT ip, device_id FROM addresses WHERE pool_id = ?`, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query addresses: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ip, deviceID string
		if err := rows.Scan(&ip, &deviceID); err != nil {
			return nil, fmt.Errorf("failed to scan address: %w", err)
		}
		addressMap[ip] = deviceID
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get all active reservations in this pool
	reservationMap := make(map[string]string) // ip -> reservation_id
	resRows, err := s.db.QueryContext(ctx, `
		SELECT ip_address, id FROM reservations
		WHERE pool_id = ? AND status = ?
	`, poolID, "active")
	if err != nil {
		return nil, fmt.Errorf("failed to query reservations: %w", err)
	}
	defer resRows.Close()

	for resRows.Next() {
		var ip, reservationID string
		if err := resRows.Scan(&ip, &reservationID); err != nil {
			return nil, fmt.Errorf("failed to scan reservation: %w", err)
		}
		reservationMap[ip] = reservationID
	}
	if err := resRows.Err(); err != nil {
		return nil, err
	}

	// Build heatmap
	var heatmap []IPStatus
	current := make(net.IP, len(startIP))
	copy(current, startIP)

	// Safety limit: don't enumerate more than 65536 IPs to prevent memory issues
	const maxIPs = 65536
	count := 0

	for count < maxIPs {
		ipStr := current.String()
		status := IPStatus{
			IP:     ipStr,
			Status: "available",
		}

		if deviceID, exists := addressMap[ipStr]; exists {
			status.Status = "used"
			status.DeviceID = deviceID
		} else if _, reserved := reservationMap[ipStr]; reserved {
			status.Status = "reserved"
		}

		heatmap = append(heatmap, status)
		count++

		// Check if we've reached the end
		if current.Equal(endIP) {
			break
		}

		// Increment IP
		if !incrementIP(current, endIP) {
			break
		}
	}

	if heatmap == nil {
		heatmap = []IPStatus{}
	}

	return heatmap, nil
}
