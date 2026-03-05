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

// Network operations

// ListNetworks retrieves all networks matching the filter criteria
func (s *SQLiteStorage) ListNetworks(ctx context.Context, filter *model.NetworkFilter) ([]model.Network, error) {

	query := `SELECT id, name, subnet, vlan_id, datacenter_id, description, created_at, updated_at FROM networks`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.Name != "" {
			conditions = append(conditions, "name LIKE ?")
			args = append(args, "%"+filter.Name+"%")
		}
		if filter.DatacenterID != "" {
			conditions = append(conditions, "datacenter_id = ?")
			args = append(args, filter.DatacenterID)
		}
		if filter.VLANID > 0 {
			conditions = append(conditions, "vlan_id = ?")
			args = append(args, filter.VLANID)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}
	defer rows.Close()

	var networks []model.Network
	for rows.Next() {
		var network model.Network
		var vlanID sql.NullInt64
		var datacenterID sql.NullString
		if err := rows.Scan(
			&network.ID, &network.Name, &network.Subnet, &vlanID,
			&datacenterID, &network.Description, &network.CreatedAt, &network.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan network: %w", err)
		}
		if vlanID.Valid {
			network.VLANID = int(vlanID.Int64)
		}
		if datacenterID.Valid {
			network.DatacenterID = datacenterID.String
		}
		networks = append(networks, network)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if networks == nil {
		networks = []model.Network{}
	}

	return networks, nil
}

// SearchNetworks performs a full-text search across network fields using FTS5
func (s *SQLiteStorage) SearchNetworks(ctx context.Context, query string) ([]model.Network, error) {
	if query == "" {
		return s.ListNetworks(ctx, nil)
	}

	ftsQuery := escapeFTSQuery(query)

	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.name, n.subnet, n.vlan_id, n.datacenter_id, n.description,
		       n.created_at, n.updated_at
		FROM networks n
		INNER JOIN networks_fts fts ON n.id = fts.id
		WHERE networks_fts MATCH ?
		ORDER BY n.name
	`, ftsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search networks: %w", err)
	}
	defer rows.Close()

	var networks []model.Network
	for rows.Next() {
		var network model.Network
		var vlanID sql.NullInt64
		var datacenterID sql.NullString
		if err := rows.Scan(
			&network.ID, &network.Name, &network.Subnet, &vlanID,
			&datacenterID, &network.Description, &network.CreatedAt, &network.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan network: %w", err)
		}
		if vlanID.Valid {
			network.VLANID = int(vlanID.Int64)
		}
		if datacenterID.Valid {
			network.DatacenterID = datacenterID.String
		}
		networks = append(networks, network)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if networks == nil {
		networks = []model.Network{}
	}

	return networks, nil
}

// GetNetwork retrieves a network by ID
func (s *SQLiteStorage) GetNetwork(ctx context.Context, id string) (*model.Network, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	network := &model.Network{}
	var vlanID sql.NullInt64
	var datacenterID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, subnet, vlan_id, datacenter_id, description, created_at, updated_at
		FROM networks WHERE id = ?
	`, id).Scan(
		&network.ID, &network.Name, &network.Subnet, &vlanID,
		&datacenterID, &network.Description, &network.CreatedAt, &network.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNetworkNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get network: %w", err)
	}

	if vlanID.Valid {
		network.VLANID = int(vlanID.Int64)
	}
	if datacenterID.Valid {
		network.DatacenterID = datacenterID.String
	}

	return network, nil
}

// CreateNetwork creates a new network
func (s *SQLiteStorage) CreateNetwork(ctx context.Context, network *model.Network) error {
	if network == nil {
		return fmt.Errorf("network is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.createNetworkInTx(ctx, tx, network); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "create", "network", network.ID, network)
	return nil
}

// createNetworkInTx creates a network within an existing transaction
func (s *SQLiteStorage) createNetworkInTx(ctx context.Context, tx *sql.Tx, network *model.Network) error {

	// Generate ID if not provided
	if network.ID == "" {
		network.ID = newUUID()
	}

	now := time.Now().UTC()
	network.CreatedAt = now
	network.UpdatedAt = now

	_, err := tx.ExecContext(ctx, `
		INSERT INTO networks (id, name, subnet, vlan_id, datacenter_id, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, network.ID, network.Name, network.Subnet, nullInt(network.VLANID),
		nullString(network.DatacenterID), network.Description, network.CreatedAt, network.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	return nil
}

// UpdateNetwork updates an existing network
func (s *SQLiteStorage) UpdateNetwork(ctx context.Context, network *model.Network) error {
	if network == nil {
		return fmt.Errorf("network is nil")
	}
	if network.ID == "" {
		return ErrInvalidID
	}

	// Check if network exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM networks WHERE id = ?)`, network.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check network existence: %w", err)
	}
	if !exists {
		return ErrNetworkNotFound
	}

	network.UpdatedAt = time.Now().UTC()

	_, err = s.db.ExecContext(ctx, `
		UPDATE networks SET name = ?, subnet = ?, vlan_id = ?, datacenter_id = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, network.Name, network.Subnet, nullInt(network.VLANID),
		nullString(network.DatacenterID), network.Description, network.UpdatedAt, network.ID)

	if err != nil {
		return fmt.Errorf("failed to update network: %w", err)
	}

	s.auditLog(ctx, "update", "network", network.ID, network)
	return nil
}

// DeleteNetwork removes a network by ID
func (s *SQLiteStorage) DeleteNetwork(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.deleteNetworkInTx(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "delete", "network", id, nil)
	return nil
}

// deleteNetworkInTx deletes a network within an existing transaction
func (s *SQLiteStorage) deleteNetworkInTx(ctx context.Context, tx *sql.Tx, id string) error {

	// Check if network exists
	var exists bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM networks WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check network existence: %w", err)
	}
	if !exists {
		return ErrNetworkNotFound
	}

	// Unlink addresses from this network (set network_id to NULL)
	_, err = tx.ExecContext(ctx, `UPDATE addresses SET network_id = NULL WHERE network_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to unlink addresses: %w", err)
	}

	// Delete network pools (cascades via foreign key, but explicit for clarity)
	_, err = tx.ExecContext(ctx, `DELETE FROM network_pools WHERE network_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete network pools: %w", err)
	}

	// Delete the network
	_, err = tx.ExecContext(ctx, `DELETE FROM networks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	return nil
}

// GetNetworkDevices retrieves all devices that have addresses in a network
func (s *SQLiteStorage) GetNetworkDevices(ctx context.Context, networkID string) ([]model.Device, error) {
	if networkID == "" {
		return nil, ErrInvalidID
	}

	// Check if network exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM networks WHERE id = ?)`, networkID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check network existence: %w", err)
	}
	if !exists {
		return nil, ErrNetworkNotFound
	}

	// Use ListDevices with a filter
	return s.ListDevices(ctx, &model.DeviceFilter{NetworkID: networkID})
}

// GetNetworkUtilization calculates IP usage for a network based on its CIDR and assigned addresses
func (s *SQLiteStorage) GetNetworkUtilization(ctx context.Context, networkID string) (*model.NetworkUtilization, error) {
	if networkID == "" {
		return nil, ErrInvalidID
	}

	// Get the network to retrieve its subnet
	network, err := s.GetNetwork(ctx, networkID)
	if err != nil {
		return nil, err
	}

	// Calculate total IPs from CIDR
	totalIPs, err := calculateCIDRSize(network.Subnet)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subnet CIDR: %w", err)
	}

	// Count used IPs (addresses assigned to this network)
	var usedIPs int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT ip) FROM addresses WHERE network_id = ?
	`, networkID).Scan(&usedIPs)
	if err != nil {
		return nil, fmt.Errorf("failed to count used IPs: %w", err)
	}

	availableIPs := max(totalIPs-usedIPs, 0)

	var utilization float64
	if totalIPs > 0 {
		utilization = float64(usedIPs) / float64(totalIPs) * 100
	}

	return &model.NetworkUtilization{
		NetworkID:    networkID,
		TotalIPs:     totalIPs,
		UsedIPs:      usedIPs,
		AvailableIPs: availableIPs,
		Utilization:  utilization,
	}, nil
}

// calculateCIDRSize calculates the number of usable host IPs in a CIDR block
func calculateCIDRSize(cidr string) (int, error) {
	// Parse CIDR (e.g., "192.168.1.0/24")
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, err
	}

	// Get the prefix length
	ones, bits := ipNet.Mask.Size()
	if bits == 0 {
		return 0, fmt.Errorf("invalid mask")
	}

	// Calculate total addresses: 2^(bits - ones)
	hostBits := bits - ones
	if hostBits <= 0 {
		return 1, nil // /32 or /128 has 1 address
	}

	// For large subnets, cap at a reasonable number
	if hostBits > 20 {
		// For subnets larger than /12, return a capped value
		return 1 << 20, nil // ~1 million
	}

	total := 1 << hostBits

	// Subtract network and broadcast addresses for IPv4 subnets with more than 2 hosts
	if bits == 32 && hostBits >= 2 {
		total -= 2 // Subtract network and broadcast addresses
	}

	if total < 0 {
		total = 0
	}

	return total, nil
}
