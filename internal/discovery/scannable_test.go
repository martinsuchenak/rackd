package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestCheckScannable(t *testing.T) {
	for subnet, ok := range map[string]bool{
		"10.0.0.0/24":        true,
		"192.168.1.0/24":     true,
		"203.0.113.0/24":     true,
		"2001:db8::/120":     true,
		"127.0.0.0/24":       false,
		"169.254.169.254/32": false, // cloud metadata
		"169.254.0.0/16":     false,
		"0.0.0.0/24":         false,
		"224.0.0.0/24":       false,
		"::1/128":            false,
		"fe80::/120":         false,
		"::ffff:7f00:0/120":  false, // 127.0.0.0 mapped into IPv6
	} {
		err := checkScannable(subnet)
		var notScannable *SubnetNotScannableError
		if ok && err != nil {
			t.Errorf("%s: %v, want scannable", subnet, err)
		}
		if !ok && !errors.As(err, &notScannable) {
			t.Errorf("%s: %v, want SubnetNotScannableError", subnet, err)
		}
	}
}

// Both refusals happen before a scan record is created, so the scanner needs
// no storage here.
func TestScanRefusesBeforeStarting(t *testing.T) {
	s := &UnifiedScanner{scans: map[string]*model.DiscoveryScan{}}
	_, err := s.ScanWithOptions(context.Background(), &model.Network{ID: "n", Subnet: "127.0.0.0/24"}, &ScanOptions{ScanType: model.ScanTypeQuick})
	var notScannable *SubnetNotScannableError
	if !errors.As(err, &notScannable) {
		t.Errorf("loopback scan: %v", err)
	}

	s.active = MaxConcurrentScans
	_, err = s.ScanWithOptions(context.Background(), &model.Network{ID: "n", Subnet: "10.0.0.0/24"}, &ScanOptions{ScanType: model.ScanTypeQuick})
	if !errors.Is(err, ErrTooManyScans) {
		t.Errorf("scan beyond the limit: %v", err)
	}
	if s.active != MaxConcurrentScans {
		t.Errorf("a refused scan changed the active count to %d", s.active)
	}
}
