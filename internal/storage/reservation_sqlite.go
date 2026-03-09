package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Reservation operations

// CreateReservation creates a new IP reservation
func (s *SQLiteStorage) CreateReservation(ctx context.Context, reservation *model.Reservation) error {
	if reservation == nil {
		return fmt.Errorf("reservation is nil")
	}

	// Generate ID if not provided
	if reservation.ID == "" {
		reservation.ID = newUUID()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate pool exists and get IP range for validation
	var startIP, endIP string
	err = tx.QueryRowContext(ctx, `SELECT start_ip, end_ip FROM network_pools WHERE id = ?`, reservation.PoolID).Scan(&startIP, &endIP)
	if err == sql.ErrNoRows {
		return ErrPoolNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check pool existence: %w", err)
	}

	// Validate IP is in pool range
	checkIP := net.ParseIP(reservation.IPAddress)
	poolStartIP := net.ParseIP(startIP)
	poolEndIP := net.ParseIP(endIP)
	if checkIP == nil || poolStartIP == nil || poolEndIP == nil {
		return fmt.Errorf("invalid IP address")
	}
	checkIP = checkIP.To4()
	poolStartIP = poolStartIP.To4()
	poolEndIP = poolEndIP.To4()
	if checkIP == nil || poolStartIP == nil || poolEndIP == nil {
		return fmt.Errorf("only IPv4 addresses are currently supported")
	}
	if !ipInRange(checkIP, poolStartIP, poolEndIP) {
		return fmt.Errorf("IP address %s is not within pool range", reservation.IPAddress)
	}

	// Check if IP is already used by a device
	var usedByDevice bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM addresses WHERE pool_id = ? AND ip = ?)`,
		reservation.PoolID, reservation.IPAddress).Scan(&usedByDevice)
	if err != nil {
		return fmt.Errorf("failed to check IP usage: %w", err)
	}
	if usedByDevice {
		return ErrIPConflict
	}

	// Check if IP is already reserved
	var reservedCount int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reservations
		WHERE pool_id = ? AND ip_address = ? AND status = ?
	`, reservation.PoolID, reservation.IPAddress, string(model.ReservationStatusActive)).Scan(&reservedCount)
	if err != nil {
		return fmt.Errorf("failed to check reservation: %w", err)
	}
	if reservedCount > 0 {
		return ErrIPAlreadyReserved
	}

	now := nowUTC()
	if reservation.ReservedAt.IsZero() {
		reservation.ReservedAt = now
	}
	reservation.CreatedAt = now
	reservation.UpdatedAt = now

	// Set default status
	if reservation.Status == "" {
		reservation.Status = model.ReservationStatusActive
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO reservations (
			id, pool_id, ip_address, hostname, purpose, reserved_by, reserved_at,
			expires_at, status, notes, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, reservation.ID, reservation.PoolID, reservation.IPAddress, nullString(reservation.Hostname),
		nullString(reservation.Purpose), reservation.ReservedBy, reservation.ReservedAt,
		nullTime(reservation.ExpiresAt), string(reservation.Status), nullString(reservation.Notes),
		reservation.CreatedAt, reservation.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrIPAlreadyReserved
		}
		return fmt.Errorf("failed to create reservation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "create", "reservation", reservation.ID, reservation)
	return nil
}

// GetReservation retrieves a reservation by ID
func (s *SQLiteStorage) GetReservation(ctx context.Context, id string) (*model.Reservation, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	var reservation model.Reservation
	var hostname, purpose, notes sql.NullString
	var expiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, pool_id, ip_address, hostname, purpose, reserved_by, reserved_at,
		       expires_at, status, notes, created_at, updated_at
		FROM reservations WHERE id = ?
	`, id).Scan(
		&reservation.ID, &reservation.PoolID, &reservation.IPAddress, &hostname, &purpose,
		&reservation.ReservedBy, &reservation.ReservedAt, &expiresAt, &reservation.Status,
		&notes, &reservation.CreatedAt, &reservation.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrReservationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}

	if hostname.Valid {
		reservation.Hostname = hostname.String
	}
	if purpose.Valid {
		reservation.Purpose = purpose.String
	}
	if notes.Valid {
		reservation.Notes = notes.String
	}
	if expiresAt.Valid {
		reservation.ExpiresAt = &expiresAt.Time
	}

	return &reservation, nil
}

// GetReservationByIP retrieves a reservation by pool ID and IP address
func (s *SQLiteStorage) GetReservationByIP(ctx context.Context, poolID, ip string) (*model.Reservation, error) {
	if poolID == "" || ip == "" {
		return nil, ErrInvalidID
	}

	var reservation model.Reservation
	var hostname, purpose, notes sql.NullString
	var expiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, pool_id, ip_address, hostname, purpose, reserved_by, reserved_at,
		       expires_at, status, notes, created_at, updated_at
		FROM reservations WHERE pool_id = ? AND ip_address = ? AND status = ?
	`, poolID, ip, string(model.ReservationStatusActive)).Scan(
		&reservation.ID, &reservation.PoolID, &reservation.IPAddress, &hostname, &purpose,
		&reservation.ReservedBy, &reservation.ReservedAt, &expiresAt, &reservation.Status,
		&notes, &reservation.CreatedAt, &reservation.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrReservationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reservation by IP: %w", err)
	}

	if hostname.Valid {
		reservation.Hostname = hostname.String
	}
	if purpose.Valid {
		reservation.Purpose = purpose.String
	}
	if notes.Valid {
		reservation.Notes = notes.String
	}
	if expiresAt.Valid {
		reservation.ExpiresAt = &expiresAt.Time
	}

	return &reservation, nil
}

// ListReservations retrieves reservations matching filter criteria
func (s *SQLiteStorage) ListReservations(ctx context.Context, filter *model.ReservationFilter) ([]model.Reservation, error) {

	query := `SELECT id, pool_id, ip_address, hostname, purpose, reserved_by, reserved_at,
	          expires_at, status, notes, created_at, updated_at
	          FROM reservations`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.PoolID != "" {
			conditions = append(conditions, "pool_id = ?")
			args = append(args, filter.PoolID)
		}
		if filter.Status != "" {
			conditions = append(conditions, "status = ?")
			args = append(args, string(filter.Status))
		}
		if filter.ReservedBy != "" {
			conditions = append(conditions, "reserved_by = ?")
			args = append(args, filter.ReservedBy)
		}
		if filter.IPAddress != "" {
			conditions = append(conditions, "ip_address = ?")
			args = append(args, filter.IPAddress)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY reserved_at DESC"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list reservations: %w", err)
	}
	defer rows.Close()

	return scanReservations(rows)
}

// UpdateReservation updates an existing reservation
func (s *SQLiteStorage) UpdateReservation(ctx context.Context, reservation *model.Reservation) error {
	if reservation == nil {
		return fmt.Errorf("reservation is nil")
	}
	if reservation.ID == "" {
		return ErrInvalidID
	}

	reservation.UpdatedAt = nowUTC()

	_, err := s.db.ExecContext(ctx, `
		UPDATE reservations SET
			hostname = ?, purpose = ?, expires_at = ?, status = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, nullString(reservation.Hostname), nullString(reservation.Purpose),
		nullTime(reservation.ExpiresAt), string(reservation.Status),
		nullString(reservation.Notes), reservation.UpdatedAt, reservation.ID)

	if err != nil {
		return fmt.Errorf("failed to update reservation: %w", err)
	}

	s.auditLog(ctx, "update", "reservation", reservation.ID, reservation)
	return nil
}

// DeleteReservation removes a reservation by ID
func (s *SQLiteStorage) DeleteReservation(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	_, err := s.db.ExecContext(ctx, `DELETE FROM reservations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete reservation: %w", err)
	}

	s.auditLog(ctx, "delete", "reservation", id, nil)
	return nil
}

// GetReservationsByPool retrieves all reservations for a specific pool
func (s *SQLiteStorage) GetReservationsByPool(ctx context.Context, poolID string) ([]model.Reservation, error) {
	if poolID == "" {
		return nil, ErrInvalidID
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, pool_id, ip_address, hostname, purpose, reserved_by, reserved_at,
		       expires_at, status, notes, created_at, updated_at
		FROM reservations WHERE pool_id = ? AND status = ?
		ORDER BY ip_address
	`, poolID, string(model.ReservationStatusActive))
	if err != nil {
		return nil, fmt.Errorf("failed to get reservations by pool: %w", err)
	}
	defer rows.Close()

	return scanReservations(rows)
}

// GetReservationsByUser retrieves all reservations made by a specific user
func (s *SQLiteStorage) GetReservationsByUser(ctx context.Context, userID string) ([]model.Reservation, error) {
	if userID == "" {
		return nil, ErrInvalidID
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, pool_id, ip_address, hostname, purpose, reserved_by, reserved_at,
		       expires_at, status, notes, created_at, updated_at
		FROM reservations WHERE reserved_by = ?
		ORDER BY reserved_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reservations by user: %w", err)
	}
	defer rows.Close()

	return scanReservations(rows)
}

// ExpireReservations marks all expired reservations as expired
func (s *SQLiteStorage) ExpireReservations(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE reservations SET status = ?, updated_at = ?
		WHERE status = ? AND expires_at IS NOT NULL AND expires_at < ?
	`, string(model.ReservationStatusExpired), nowUTC(),
		string(model.ReservationStatusActive), nowUTC())
	if err != nil {
		return 0, fmt.Errorf("failed to expire reservations: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return affected, nil
}

// IsIPReserved checks if an IP address is reserved in a pool
func (s *SQLiteStorage) IsIPReserved(ctx context.Context, poolID, ip string) (bool, error) {
	if poolID == "" || ip == "" {
		return false, ErrInvalidID
	}

	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reservations
		WHERE pool_id = ? AND ip_address = ? AND status = ?
	`, poolID, ip, string(model.ReservationStatusActive)).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check IP reservation: %w", err)
	}

	return count > 0, nil
}

// scanReservations is a helper to scan reservation rows
func scanReservations(rows *sql.Rows) ([]model.Reservation, error) {
	var reservations []model.Reservation
	for rows.Next() {
		var reservation model.Reservation
		var hostname, purpose, notes sql.NullString
		var expiresAt sql.NullTime

		if err := rows.Scan(
			&reservation.ID, &reservation.PoolID, &reservation.IPAddress, &hostname, &purpose,
			&reservation.ReservedBy, &reservation.ReservedAt, &expiresAt, &reservation.Status,
			&notes, &reservation.CreatedAt, &reservation.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan reservation: %w", err)
		}

		if hostname.Valid {
			reservation.Hostname = hostname.String
		}
		if purpose.Valid {
			reservation.Purpose = purpose.String
		}
		if notes.Valid {
			reservation.Notes = notes.String
		}
		if expiresAt.Valid {
			reservation.ExpiresAt = &expiresAt.Time
		}

		reservations = append(reservations, reservation)
	}

	if reservations == nil {
		reservations = []model.Reservation{}
	}

	return reservations, nil
}
