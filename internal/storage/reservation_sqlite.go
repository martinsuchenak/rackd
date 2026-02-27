package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

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

	// Validate pool exists
	var poolExists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM network_pools WHERE id = ?)`, reservation.PoolID).Scan(&poolExists)
	if err != nil {
		return fmt.Errorf("failed to check pool existence: %w", err)
	}
	if !poolExists {
		return ErrPoolNotFound
	}

	// Validate IP is in pool range
	inRange, err := s.ValidateIPInPool(reservation.PoolID, reservation.IPAddress)
	if err != nil {
		return fmt.Errorf("failed to validate IP in pool: %w", err)
	}
	if !inRange {
		return fmt.Errorf("IP address %s is not within pool range", reservation.IPAddress)
	}

	// Check if IP is already used by a device
	var usedByDevice bool
	err = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM addresses WHERE pool_id = ? AND ip = ?)`,
		reservation.PoolID, reservation.IPAddress).Scan(&usedByDevice)
	if err != nil {
		return fmt.Errorf("failed to check IP usage: %w", err)
	}
	if usedByDevice {
		return ErrIPConflict
	}

	// Check if IP is already reserved
	isReserved, err := s.IsIPReserved(reservation.PoolID, reservation.IPAddress)
	if err != nil {
		return fmt.Errorf("failed to check reservation: %w", err)
	}
	if isReserved {
		return ErrIPAlreadyReserved
	}

	now := time.Now().UTC()
	if reservation.ReservedAt.IsZero() {
		reservation.ReservedAt = now
	}
	reservation.CreatedAt = now
	reservation.UpdatedAt = now

	// Set default status
	if reservation.Status == "" {
		reservation.Status = model.ReservationStatusActive
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO reservations (
			id, pool_id, ip_address, hostname, purpose, reserved_by, reserved_at,
			expires_at, status, notes, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, reservation.ID, reservation.PoolID, reservation.IPAddress, nullString(reservation.Hostname),
		nullString(reservation.Purpose), reservation.ReservedBy, reservation.ReservedAt,
		nullTime(reservation.ExpiresAt), string(reservation.Status), nullString(reservation.Notes),
		reservation.CreatedAt, reservation.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create reservation: %w", err)
	}

	s.auditLog(ctx, "create", "reservation", reservation.ID, reservation)
	return nil
}

// GetReservation retrieves a reservation by ID
func (s *SQLiteStorage) GetReservation(id string) (*model.Reservation, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

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
func (s *SQLiteStorage) GetReservationByIP(poolID, ip string) (*model.Reservation, error) {
	if poolID == "" || ip == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

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
func (s *SQLiteStorage) ListReservations(filter *model.ReservationFilter) ([]model.Reservation, error) {
	ctx := context.Background()

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

	reservation.UpdatedAt = time.Now().UTC()

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
func (s *SQLiteStorage) GetReservationsByPool(poolID string) ([]model.Reservation, error) {
	if poolID == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

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
func (s *SQLiteStorage) GetReservationsByUser(userID string) ([]model.Reservation, error) {
	if userID == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

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
	`, string(model.ReservationStatusExpired), time.Now().UTC(),
		string(model.ReservationStatusActive), time.Now().UTC())
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
func (s *SQLiteStorage) IsIPReserved(poolID, ip string) (bool, error) {
	if poolID == "" || ip == "" {
		return false, ErrInvalidID
	}

	ctx := context.Background()

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
