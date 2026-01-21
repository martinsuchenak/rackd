package discovery

import (
	"context"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Scanner interface for network discovery
type Scanner interface {
	Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error)
	GetScanStatus(scanID string) (*model.DiscoveryScan, error)
}
