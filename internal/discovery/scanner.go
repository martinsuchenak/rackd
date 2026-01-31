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
	storage     storage.DiscoveryStorage
	config      *config.Config
	scans       map[string]*model.DiscoveryScan
	cancelFuncs map[string]context.CancelFunc
	mu          sync.RWMutex
}

func NewScanner(store storage.DiscoveryStorage, cfg *config.Config) *DefaultScanner {
	return &DefaultScanner{
		storage:     store,
		config:      cfg,
		scans:       make(map[string]*model.DiscoveryScan),
		cancelFuncs: make(map[string]context.CancelFunc),
	}
}

// MaxSubnetBits is the maximum subnet size allowed for scanning (default /16 = 65536 hosts)
const MaxSubnetBits = 16

var ErrSubnetTooLarge = fmt.Errorf("subnet too large: maximum /%d allowed", 32-MaxSubnetBits)
var ErrScanNotFound = fmt.Errorf("scan not found")
var ErrScanNotRunning = fmt.Errorf("scan is not running or pending")

func (s *DefaultScanner) Scan(scanCtx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
	_, ipNet, err := net.ParseCIDR(network.Subnet)
	if err != nil {
		log.Error("Failed to parse CIDR", "subnet", network.Subnet, "error", err)
		return nil, err
	}

	ones, bits := ipNet.Mask.Size()
	if bits-ones > MaxSubnetBits {
		log.Error("Subnet too large", "subnet", network.Subnet, "ones", ones, "bits", bits)
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
		log.Error("Failed to create scan in database", "scan_id", scan.ID, "error", err)
		return nil, err
	}

	// Create cancellable context for the scan
	scanCtxCancellable, cancel := context.WithCancel(context.Background())

	s.mu.Lock()
	s.scans[scan.ID] = scan
	s.cancelFuncs[scan.ID] = cancel
	s.mu.Unlock()

	log.Info("Starting discovery scan", "network", network.Name, "network_id", network.ID, "scan_id", scan.ID, "scan_type", scanType, "hosts", scan.TotalHosts)

	go func() {
		defer func() {
			// Clean up cancel function when scan completes
			s.mu.Lock()
			delete(s.cancelFuncs, scan.ID)
			s.mu.Unlock()

			if r := recover(); r != nil {
				log.Error("Panic in scan goroutine", "scan_id", scan.ID, "panic", r)
				now := time.Now()
				scan.Status = model.ScanStatusFailed
				scan.ErrorMessage = "panic during scan initialization"
				scan.CompletedAt = &now
				s.storage.UpdateDiscoveryScan(scan)
			}
		}()

		// Check if context was cancelled before starting (e.g., server shutdown)
		select {
		case <-scanCtx.Done():
			log.Info("Scan cancelled before starting", "scan_id", scan.ID)
			scan.Status = model.ScanStatusFailed
			scan.ErrorMessage = "scan cancelled before starting"
			s.storage.UpdateDiscoveryScan(scan)
			return
		default:
		}

		s.runScan(scanCtxCancellable, scan, network, ipNet, scanType)
	}()

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

func (s *DefaultScanner) CancelScan(scanID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[scanID]
	if !ok {
		return ErrScanNotFound
	}

	if scan.Status != model.ScanStatusRunning && scan.Status != model.ScanStatusPending {
		return ErrScanNotRunning
	}

	cancel, ok := s.cancelFuncs[scanID]
	if !ok {
		return ErrScanNotFound
	}

	log.Info("Cancelling scan", "scan_id", scanID)
	cancel()
	delete(s.cancelFuncs, scanID)

	return nil
}

func (s *DefaultScanner) runScan(ctx context.Context, scan *model.DiscoveryScan, network *model.Network, ipNet *net.IPNet, scanType string) {
	log.Info("runScan started", "scan_id", scan.ID, "scan_type", scanType)

	now := time.Now()
	scan.Status = model.ScanStatusRunning
	scan.StartedAt = &now
	if err := s.storage.UpdateDiscoveryScan(scan); err != nil {
		log.Error("Failed to update scan to running", "scan_id", scan.ID, "error", err)
		return
	}
	log.Info("Scan status updated to running", "scan_id", scan.ID)

	ips := expandCIDR(ipNet)
	scan.TotalHosts = len(ips)

	log.Info("IPs expanded", "scan_id", scan.ID, "total_hosts", scan.TotalHosts, "scan_type", scanType)

	semaphore := make(chan struct{}, s.config.DiscoveryMaxConcurrent)
	var wg sync.WaitGroup
	var foundCount int
	var mu sync.Mutex

	log.Info("Starting host scanning loop", "scan_id", scan.ID, "max_concurrent", s.config.DiscoveryMaxConcurrent, "timeout", s.config.DiscoveryTimeout)

	for i, ip := range ips {
		select {
		case <-ctx.Done():
			log.Info("Scan cancelled by context", "scan_id", scan.ID)
			scan.Status = model.ScanStatusFailed
			scan.ErrorMessage = "scan cancelled"
			if err := s.storage.UpdateDiscoveryScan(scan); err != nil {
				log.Error("Failed to update scan to cancelled", "scan_id", scan.ID, "error", err)
			}
			return
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(ip string, index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			if s.isHostAlive(ip) {
				log.Debug("Host is alive", "scan_id", scan.ID, "ip", ip, "index", index)
				device := s.discoverHost(ip, network.ID, scanType)
				if device != nil {
					existing, _ := s.storage.GetDiscoveredDeviceByIP(network.ID, ip)
					if existing != nil {
						device.ID = existing.ID
						device.FirstSeen = existing.FirstSeen
						if err := s.storage.UpdateDiscoveredDevice(device); err != nil {
							log.Error("Failed to update discovered device", "ip", ip, "error", err)
						}
					} else {
						if err := s.storage.CreateDiscoveredDevice(device); err != nil {
							log.Error("Failed to create discovered device", "ip", ip, "error", err)
						}
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
			if err := s.storage.UpdateDiscoveryScan(scan); err != nil {
				log.Error("Failed to update scan progress", "scan_id", scan.ID, "ip", ip, "scanned", scan.ScannedHosts, "error", err)
			}
		}(ip, i)
	}

	log.Info("Waiting for all goroutines to complete", "scan_id", scan.ID)
	wg.Wait()

	completedAt := time.Now()
	scan.Status = model.ScanStatusCompleted
	scan.CompletedAt = &completedAt
	if err := s.storage.UpdateDiscoveryScan(scan); err != nil {
		log.Error("Failed to update scan to completed", "scan_id", scan.ID, "error", err)
	}

	s.cleanupCompletedScans()

	log.Info("Discovery scan completed", "network", network.Name, "found", scan.FoundHosts, "scanned", scan.ScannedHosts, "duration", completedAt.Sub(now).String())
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
			log.Debug("Host alive", "ip", ip, "port", port)
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

	// DNS lookup with timeout
	lookupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	type lookupResult struct {
		names []string
		err   error
	}
	resultCh := make(chan lookupResult, 1)

	go func() {
		names, err := net.LookupAddr(ip)
		resultCh <- lookupResult{names: names, err: err}
	}()

	select {
	case <-lookupCtx.Done():
		log.Warn("DNS lookup timeout", "ip", ip)
	case result := <-resultCh:
		if result.err == nil && len(result.names) > 0 {
			device.Hostname = result.names[0]
		} else if result.err != nil {
			log.Debug("DNS lookup failed", "ip", ip, "error", result.err)
		}
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
