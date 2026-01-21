package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type DefaultScanner struct {
	storage storage.DiscoveryStorage
	config  *config.Config
	scans   map[string]*model.DiscoveryScan
	mu      sync.RWMutex
}

func NewScanner(store storage.DiscoveryStorage, cfg *config.Config) *DefaultScanner {
	return &DefaultScanner{
		storage: store,
		config:  cfg,
		scans:   make(map[string]*model.DiscoveryScan),
	}
}

// MaxSubnetBits is the maximum subnet size allowed for scanning (default /16 = 65536 hosts)
const MaxSubnetBits = 16

var ErrSubnetTooLarge = fmt.Errorf("subnet too large: maximum /%d allowed", 32-MaxSubnetBits)

func (s *DefaultScanner) Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
	_, ipNet, err := net.ParseCIDR(network.Subnet)
	if err != nil {
		return nil, err
	}

	ones, bits := ipNet.Mask.Size()
	if bits-ones > MaxSubnetBits {
		return nil, ErrSubnetTooLarge
	}

	scan := &model.DiscoveryScan{
		ID:         uuid.Must(uuid.NewV7()).String(),
		NetworkID:  network.ID,
		Status:     model.ScanStatusPending,
		ScanType:   scanType,
		TotalHosts: countHosts(ipNet),
	}

	if err := s.storage.CreateDiscoveryScan(scan); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.scans[scan.ID] = scan
	s.mu.Unlock()

	go s.runScan(ctx, scan, network, ipNet, scanType)

	return scan, nil
}

func (s *DefaultScanner) GetScanStatus(scanID string) (*model.DiscoveryScan, error) {
	s.mu.RLock()
	scan, ok := s.scans[scanID]
	s.mu.RUnlock()
	if ok {
		return scan, nil
	}
	return s.storage.GetDiscoveryScan(scanID)
}

func (s *DefaultScanner) runScan(ctx context.Context, scan *model.DiscoveryScan, network *model.Network, ipNet *net.IPNet, scanType string) {
	now := time.Now()
	scan.Status = model.ScanStatusRunning
	scan.StartedAt = &now
	s.storage.UpdateDiscoveryScan(scan)

	ips := expandCIDR(ipNet)
	scan.TotalHosts = len(ips)

	semaphore := make(chan struct{}, s.config.DiscoveryMaxConcurrent)
	var wg sync.WaitGroup
	var foundCount int
	var mu sync.Mutex

	for i, ip := range ips {
		select {
		case <-ctx.Done():
			scan.Status = model.ScanStatusFailed
			scan.ErrorMessage = "scan cancelled"
			s.storage.UpdateDiscoveryScan(scan)
			return
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(ip string, index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			if s.isHostAlive(ip) {
				device := s.discoverHost(ip, network.ID, scanType)
				if device != nil {
					existing, _ := s.storage.GetDiscoveredDeviceByIP(network.ID, ip)
					if existing != nil {
						device.ID = existing.ID
						device.FirstSeen = existing.FirstSeen
						s.storage.UpdateDiscoveredDevice(device)
					} else {
						s.storage.CreateDiscoveredDevice(device)
					}

					mu.Lock()
					foundCount++
					mu.Unlock()
				}
			}

			mu.Lock()
			scan.ScannedHosts = index + 1
			scan.FoundHosts = foundCount
			scan.ProgressPercent = float64(scan.ScannedHosts) / float64(scan.TotalHosts) * 100
			mu.Unlock()
			s.storage.UpdateDiscoveryScan(scan)
		}(ip, i)
	}

	wg.Wait()

	completedAt := time.Now()
	scan.Status = model.ScanStatusCompleted
	scan.CompletedAt = &completedAt
	s.storage.UpdateDiscoveryScan(scan)

	s.cleanupCompletedScans()

	log.Info("Discovery scan completed", "network", network.Name, "found", scan.FoundHosts, "scanned", scan.ScannedHosts)
}

func (s *DefaultScanner) cleanupCompletedScans() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, scan := range s.scans {
		if (scan.Status == model.ScanStatusCompleted || scan.Status == model.ScanStatusFailed) &&
			scan.CompletedAt != nil && time.Since(*scan.CompletedAt) > time.Hour {
			delete(s.scans, id)
		}
	}
}

func (s *DefaultScanner) isHostAlive(ip string) bool {
	ports := []int{22, 80, 443, 3389}
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), s.config.DiscoveryTimeout)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

func (s *DefaultScanner) discoverHost(ip string, networkID string, scanType string) *model.DiscoveredDevice {
	now := time.Now()
	device := &model.DiscoveredDevice{
		ID:        uuid.Must(uuid.NewV7()).String(),
		IP:        ip,
		NetworkID: networkID,
		Status:    "online",
		FirstSeen: now,
		LastSeen:  now,
		OpenPorts: []int{},
		Services:  []model.ServiceInfo{},
	}

	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		device.Hostname = names[0]
	}

	if scanType != model.ScanTypeQuick {
		device.OpenPorts = s.scanPorts(ip, scanType)
	}

	return device
}

func (s *DefaultScanner) scanPorts(ip string, scanType string) []int {
	var ports []int
	var portsToScan []int

	if scanType == model.ScanTypeFull {
		portsToScan = []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 8080}
	} else {
		portsToScan = getTop100Ports()
	}

	for _, port := range portsToScan {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Second)
		if err == nil {
			conn.Close()
			ports = append(ports, port)
		}
	}

	return ports
}

func countHosts(ipNet *net.IPNet) int {
	ones, bits := ipNet.Mask.Size()
	return 1 << (bits - ones)
}

func expandCIDR(ipNet *net.IPNet) []string {
	var ips []string
	ip := make(net.IP, len(ipNet.IP))
	copy(ip, ipNet.IP)
	ip = ip.Mask(ipNet.Mask)

	for ipNet.Contains(ip) {
		ips = append(ips, ip.String())
		incrementIP(ip)
	}

	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}
	return ips
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

func getTop100Ports() []int {
	return []int{
		7, 9, 13, 21, 22, 23, 25, 26, 37, 53, 79, 80, 81, 88, 106, 110, 111, 113, 119, 135,
		139, 143, 144, 179, 199, 389, 427, 443, 444, 445, 465, 513, 514, 515, 543, 544, 548,
		554, 587, 631, 646, 873, 990, 993, 995, 1025, 1026, 1027, 1028, 1029, 1110, 1433,
		1720, 1723, 1755, 1900, 2000, 2001, 2049, 2121, 2717, 3000, 3128, 3306, 3389, 3986,
		4899, 5000, 5009, 5051, 5060, 5101, 5190, 5357, 5432, 5631, 5666, 5800, 5900, 6000,
		6001, 6646, 7070, 8000, 8008, 8009, 8080, 8081, 8443, 8888, 9100, 9999, 10000, 32768,
		49152, 49153, 49154, 49155, 49156, 49157,
	}
}
