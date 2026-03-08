package model

import "time"

// ReservationStatus represents the status of a reservation
type ReservationStatus string

const (
	ReservationStatusActive   ReservationStatus = "active"
	ReservationStatusExpired  ReservationStatus = "expired"
	ReservationStatusClaimed  ReservationStatus = "claimed"
	ReservationStatusReleased ReservationStatus = "released"
)

// Reservation represents an IP address reservation within a pool
type Reservation struct {
	ID          string            `json:"id"`
	PoolID      string            `json:"pool_id"`
	IPAddress   string            `json:"ip_address"`
	Hostname    string            `json:"hostname,omitempty"`
	Purpose     string            `json:"purpose,omitempty"`
	ReservedBy  string            `json:"reserved_by"`
	ReservedAt  time.Time         `json:"reserved_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	Status      ReservationStatus `json:"status"`
	Notes       string            `json:"notes,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ReservationFilter defines filter criteria for listing reservations
type ReservationFilter struct {
	Pagination
	PoolID     string
	Status     ReservationStatus
	ReservedBy string
	IPAddress  string
}

// CreateReservationRequest represents a request to create a reservation
type CreateReservationRequest struct {
	PoolID    string     `json:"pool_id"`
	IPAddress string     `json:"ip_address,omitempty"` // Optional - auto-assign if not provided
	Hostname  string     `json:"hostname,omitempty"`
	Purpose   string     `json:"purpose,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Notes     string     `json:"notes,omitempty"`
}

// UpdateReservationRequest represents a request to update a reservation
type UpdateReservationRequest struct {
	Hostname  string     `json:"hostname,omitempty"`
	Purpose   string     `json:"purpose,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Notes     string     `json:"notes,omitempty"`
}

// IsExpired checks if the reservation has expired
func (r *Reservation) IsExpired() bool {
	if r.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*r.ExpiresAt)
}

// IsActive returns true if the reservation is active and not expired
func (r *Reservation) IsActive() bool {
	return r.Status == ReservationStatusActive && !r.IsExpired()
}
