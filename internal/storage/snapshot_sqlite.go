package storage

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// CreateSnapshot stores a new utilization snapshot
func (s *SQLiteStorage) CreateSnapshot(ctx context.Context, snapshot *model.UtilizationSnapshot) error {
	if snapshot.ID == "" {
		snapshot.ID = newUUID()
	}
	snapshot.CreatedAt = time.Now().UTC()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO utilization_snapshots (id, type, resource_id, resource_name, total_ips, used_ips, utilization, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, snapshot.ID, snapshot.Type, snapshot.ResourceID, snapshot.ResourceName,
		snapshot.TotalIPs, snapshot.UsedIPs, snapshot.Utilization, snapshot.Timestamp, snapshot.CreatedAt)
	return err
}

// ListSnapshots retrieves snapshots matching filter criteria
func (s *SQLiteStorage) ListSnapshots(ctx context.Context, filter *model.SnapshotFilter) ([]model.UtilizationSnapshot, error) {
	query := `SELECT id, type, resource_id, resource_name, total_ips, used_ips, utilization, timestamp, created_at
		FROM utilization_snapshots WHERE 1=1`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.Type != "" {
			conditions = append(conditions, "type = ?")
			args = append(args, filter.Type)
		}
		if filter.ResourceID != "" {
			conditions = append(conditions, "resource_id = ?")
			args = append(args, filter.ResourceID)
		}
		if filter.After != nil {
			conditions = append(conditions, "timestamp >= ?")
			args = append(args, filter.After)
		}
		if filter.Before != nil {
			conditions = append(conditions, "timestamp <= ?")
			args = append(args, filter.Before)
		}
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY timestamp DESC"

	query, args = appendPagination(query, args, &filter.Pagination)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSnapshots(rows)
}

// GetLatestSnapshots retrieves the most recent snapshot for each resource of a type
func (s *SQLiteStorage) GetLatestSnapshots(ctx context.Context, snapshotType model.SnapshotType) ([]model.UtilizationSnapshot, error) {
	query := `
		SELECT id, type, resource_id, resource_name, total_ips, used_ips, utilization, timestamp, created_at
		FROM utilization_snapshots s1
		WHERE type = ? AND timestamp = (
			SELECT MAX(timestamp) FROM utilization_snapshots s2
			WHERE s2.resource_id = s1.resource_id AND s2.type = s1.type
		)
	`

	rows, err := s.db.QueryContext(ctx, query, snapshotType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSnapshots(rows)
}

// DeleteOldSnapshots removes snapshots older than specified days
func (s *SQLiteStorage) DeleteOldSnapshots(ctx context.Context, olderThanDays int) error {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	_, err := s.db.ExecContext(ctx, `DELETE FROM utilization_snapshots WHERE timestamp < ?`, cutoff)
	return err
}

// GetUtilizationTrend retrieves utilization points for a specific resource over time
func (s *SQLiteStorage) GetUtilizationTrend(ctx context.Context, resourceType model.SnapshotType, resourceID string, days int) ([]model.UtilizationTrendPoint, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	query := `
		SELECT timestamp, utilization, used_ips
		FROM utilization_snapshots
		WHERE type = ? AND resource_id = ? AND timestamp >= ?
		ORDER BY timestamp ASC
	`

	rows, err := s.db.QueryContext(ctx, query, resourceType, resourceID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []model.UtilizationTrendPoint
	for rows.Next() {
		var t model.UtilizationTrendPoint
		if err := rows.Scan(&t.Timestamp, &t.Utilization, &t.UsedIPs); err != nil {
			return nil, err
		}
		trends = append(trends, t)
	}

	return trends, rows.Err()
}

// GetDashboardStats retrieves all dashboard statistics
func (s *SQLiteStorage) GetDashboardStats(ctx context.Context, staleDays int, recentLimit int) (*model.DashboardStats, error) {
	stats := &model.DashboardStats{
		StaleThresholdDays: staleDays,
		RecentDiscoveries:  []model.RecentDiscovery{},           // Initialize as empty slice, not nil
		NetworkUtilization: []model.NetworkUtilizationSummary{}, // Initialize as empty slice
		StaleDeviceList:    []model.StaleDevice{},               // Initialize as empty slice
	}

	// Total devices
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices`).Scan(&stats.TotalDevices)

	// Total networks
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM networks`).Scan(&stats.TotalNetworks)

	// Total pools
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_pools`).Scan(&stats.TotalPools)

	// Total datacenters
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM datacenters`).Scan(&stats.TotalDatacenters)

	// Device status counts
	s.db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN status = 'planned' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'maintenance' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'decommissioned' THEN 1 ELSE 0 END)
		FROM devices
	`).Scan(&stats.DeviceStatusCounts.Planned, &stats.DeviceStatusCounts.Active,
		&stats.DeviceStatusCounts.Maintenance, &stats.DeviceStatusCounts.Decommissioned)

	// Discovered devices count
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM discovered_devices WHERE promoted_to_device_id IS NULL`).Scan(&stats.DiscoveredDevices)

	// Stale devices (active devices not seen in discovery for X days)
	staleCutoff := time.Now().AddDate(0, 0, -staleDays)

	// Get count
	s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT d.id) FROM devices d
		WHERE d.status = 'active'
		AND NOT EXISTS (
			SELECT 1 FROM discovered_devices dd
			WHERE dd.promoted_to_device_id = d.id
			AND dd.last_seen >= ?
		)
	`, staleCutoff).Scan(&stats.StaleDevices)

	// Get list of stale devices (limit to 20 for dashboard)
	staleRows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.name, d.hostname
		FROM devices d
		WHERE d.status = 'active'
		AND NOT EXISTS (
			SELECT 1 FROM discovered_devices dd
			WHERE dd.promoted_to_device_id = d.id
			AND dd.last_seen >= ?
		)
		ORDER BY d.name
		LIMIT 20
	`, staleCutoff)
	if err == nil {
		defer staleRows.Close()
		for staleRows.Next() {
			var sd model.StaleDevice
			var hostname sql.NullString
			if err := staleRows.Scan(&sd.ID, &sd.Name, &hostname); err != nil {
				continue
			}
			sd.Hostname = hostname.String
			stats.StaleDeviceList = append(stats.StaleDeviceList, sd)
		}
	}

	// Recent discoveries
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, ip, hostname, vendor, network_id, first_seen, last_seen
		FROM discovered_devices
		WHERE promoted_to_device_id IS NULL
		ORDER BY first_seen DESC
		LIMIT ?
	`, recentLimit)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d model.RecentDiscovery
			var hostname, vendor, networkID sql.NullString
			if err := rows.Scan(&d.ID, &d.IP, &hostname, &vendor, &networkID, &d.FirstSeen, &d.LastSeen); err != nil {
				continue
			}
			d.Hostname = hostname.String
			d.Vendor = vendor.String
			d.NetworkID = networkID.String
			stats.RecentDiscoveries = append(stats.RecentDiscoveries, d)
		}
	}

	// Network utilization summary
	networks, _ := s.ListNetworks(ctx, nil)
	var totalIPs, usedIPs int
	for _, net := range networks {
		util, err := s.GetNetworkUtilization(ctx, net.ID)
		if err == nil && util != nil {
			stats.NetworkUtilization = append(stats.NetworkUtilization, model.NetworkUtilizationSummary{
				NetworkID:   net.ID,
				NetworkName: net.Name,
				Subnet:      net.Subnet,
				TotalIPs:    util.TotalIPs,
				UsedIPs:     util.UsedIPs,
				Utilization: util.Utilization,
			})
			totalIPs += util.TotalIPs
			usedIPs += util.UsedIPs
		}
	}

	// Calculate overall utilization
	if totalIPs > 0 {
		stats.OverallUtilization = float64(usedIPs) / float64(totalIPs) * 100
	}

	return stats, nil
}

// scanSnapshots helper function
func scanSnapshots(rows *sql.Rows) ([]model.UtilizationSnapshot, error) {
	var snapshots []model.UtilizationSnapshot
	for rows.Next() {
		var s model.UtilizationSnapshot
		if err := rows.Scan(&s.ID, &s.Type, &s.ResourceID, &s.ResourceName,
			&s.TotalIPs, &s.UsedIPs, &s.Utilization, &s.Timestamp, &s.CreatedAt); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}
