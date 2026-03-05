package discovery

import (
	"context"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Scanner interface for network discovery
type Scanner interface {
	Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error)
	GetScanStatus(ctx context.Context, scanID string) (*model.DiscoveryScan, error)
	CancelScan(ctx context.Context, scanID string) error
}

// AdvancedScanner interface for profile-based and credential-based discovery
type AdvancedScanner interface {
	Scanner
	GetNetwork(ctx context.Context, id string) (*model.Network, error)
	ScanAdvanced(ctx context.Context, network *model.Network, profile *model.ScanProfile, snmpCredID, sshCredID string) (*model.DiscoveryScan, error)
}
