package dns

import "context"

// Provider defines the interface for DNS providers
type Provider interface {
	// Name returns the provider type name
	Name() string

	// CreateRecord creates a new DNS record
	CreateRecord(ctx context.Context, zone string, record *Record) error

	// UpdateRecord updates an existing DNS record
	UpdateRecord(ctx context.Context, zone string, record *Record) error

	// DeleteRecord deletes a DNS record
	DeleteRecord(ctx context.Context, zone string, name string, rtype string) error

	// GetRecord retrieves a specific record
	GetRecord(ctx context.Context, zone string, name string, rtype string) (*Record, error)

	// ListRecords lists all records in a zone
	ListRecords(ctx context.Context, zone string) ([]*Record, error)

	// ListZones lists all available zones
	ListZones(ctx context.Context) ([]string, error)

	// ZoneExists checks if a zone exists
	ZoneExists(ctx context.Context, zone string) (bool, error)

	// HealthCheck verifies connectivity
	HealthCheck(ctx context.Context) error
}

// Record represents a DNS record
type Record struct {
	Name     string // Relative name (e.g., "server-01")
	Type     string // A, AAAA, CNAME, PTR, TXT
	Value    string // IP address or target
	TTL      int
	Priority *int   // For MX records
}
