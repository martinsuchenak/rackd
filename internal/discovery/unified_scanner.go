package discovery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type UnifiedScanner struct {
	storage         storage.DiscoveryStorage
	netStorage      storage.NetworkStorage
	credStore       credentials.Storage
	scans           map[string]*model.DiscoveryScan
	cancelFuncs     map[string]context.CancelFunc
	arpScanner      *ARPScanner
	snmpScanner     *SNMPScanner
	sshScanner      *SSHScanner
	bannerGrabber   *BannerGrabber
	osFingerprinter *OSFingerprinter
	ouiDatabase     *OUIDatabase
	mu              sync.RWMutex
}

func NewUnifiedScanner(store storage.DiscoveryStorage, netStore storage.NetworkStorage, credStore credentials.Storage, timeout time.Duration) *UnifiedScanner {
	arpScanner := NewARPScanner()
	arpScanner.LoadARPTable()

	return &UnifiedScanner{
		storage:         store,
		netStorage:      netStore,
		credStore:       credStore,
		scans:           make(map[string]*model.DiscoveryScan),
		cancelFuncs:     make(map[string]context.CancelFunc),
		arpScanner:      arpScanner,
		snmpScanner:     NewSNMPScanner(credStore, timeout),
		sshScanner:      NewSSHScanner(credStore, timeout),
		bannerGrabber:   NewBannerGrabber(2 * time.Second),
		osFingerprinter: NewOSFingerprinter(2 * time.Second),
		ouiDatabase:     NewOUIDatabase(),
	}
}

func (s *UnifiedScanner) GetNetwork(id string) (*model.Network, error) {
	return s.netStorage.GetNetwork(id)
}

func (s *UnifiedScanner) ScanAdvanced(ctx context.Context, network *model.Network, profile *model.ScanProfile, snmpCredID, sshCredID string) (*model.DiscoveryScan, error) {
	return s.ScanWithOptions(ctx, network, &ScanOptions{
		NetworkID:  network.ID,
		ScanType:   profile.ScanType,
		Profile:    profile,
		SSHCredID:  sshCredID,
		SNMPCredID: snmpCredID,
	})
}

type ScanOptions struct {
	NetworkID  string
	ScanType   string
	Profile    *model.ScanProfile
	SSHCredID  string
	SNMPCredID string
}

func (s *UnifiedScanner) Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
	return s.ScanWithOptions(ctx, network, &ScanOptions{
		NetworkID: network.ID,
		ScanType:  scanType,
	})
}

func (s *UnifiedScanner) ScanWithOptions(ctx context.Context, network *model.Network, opts *ScanOptions) (*model.DiscoveryScan, error) {
	_, ipNet, err := net.ParseCIDR(network.Subnet)
	if err != nil {
		return nil, err
	}

	scan := &model.DiscoveryScan{
		ID:         uuid.Must(uuid.NewV7()).String(),
		NetworkID:  network.ID,
		Status:     model.ScanStatusPending,
		ScanType:   opts.ScanType,
		TotalHosts: countHosts(ipNet),
	}

	if err := s.storage.CreateDiscoveryScan(ctx, scan); err != nil {
		return nil, err
	}

	ctxCancellable, cancel := context.WithCancel(context.Background())

	s.mu.Lock()
	s.scans[scan.ID] = scan
	s.cancelFuncs[scan.ID] = cancel
	s.mu.Unlock()

	go s.runScanWithOptions(ctxCancellable, scan, network, ipNet, opts)

	return scan, nil
}

func (s *UnifiedScanner) GetScanStatus(scanID string) (*model.DiscoveryScan, error) {
	s.mu.RLock()
	scan, ok := s.scans[scanID]
	s.mu.RUnlock()
	if ok {
		return scan, nil
	}
	return s.storage.GetDiscoveryScan(scanID)
}

func (s *UnifiedScanner) CancelScan(scanID string) error {
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

	cancel()
	delete(s.cancelFuncs, scanID)

	return nil
}

func (s *UnifiedScanner) runScanWithOptions(ctx context.Context, scan *model.DiscoveryScan, network *model.Network, ipNet *net.IPNet, opts *ScanOptions) {
	now := time.Now()
	scan.Status = model.ScanStatusRunning
	scan.StartedAt = &now
	s.storage.UpdateDiscoveryScan(ctx, scan)

	ips := expandCIDR(ipNet)
	scan.TotalHosts = len(ips)

	semaphore := make(chan struct{}, 10)
	var wg sync.WaitGroup
	var foundCount int
	var mu sync.Mutex

	for i, ip := range ips {
		select {
		case <-ctx.Done():
			scan.Status = model.ScanStatusFailed
			scan.ErrorMessage = "scan cancelled"
			s.storage.UpdateDiscoveryScan(ctx, scan)
			return
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(ip string, index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			device := s.discoverHostWithOptions(ctx, ip, network.ID, opts)
			if device != nil {
				existing, _ := s.storage.GetDiscoveredDeviceByIP(network.ID, ip)
				if existing != nil {
					device.ID = existing.ID
					device.FirstSeen = existing.FirstSeen
					s.storage.UpdateDiscoveredDevice(ctx, device)
				} else {
					s.storage.CreateDiscoveredDevice(ctx, device)
				}

				mu.Lock()
				foundCount++
				mu.Unlock()
			}

			mu.Lock()
			scan.ScannedHosts = index + 1
			scan.FoundHosts = foundCount
			scan.ProgressPercent = float64(scan.ScannedHosts) / float64(scan.TotalHosts) * 100
			mu.Unlock()
			s.storage.UpdateDiscoveryScan(ctx, scan)
		}(ip, i)
	}

	wg.Wait()

	completedAt := time.Now()
	scan.Status = model.ScanStatusCompleted
	scan.CompletedAt = &completedAt
	s.storage.UpdateDiscoveryScan(ctx, scan)

	s.cleanupCompletedScans()
}

func (s *UnifiedScanner) discoverHostWithOptions(ctx context.Context, ip string, networkID string, opts *ScanOptions) *model.DiscoveredDevice {
	if !s.isHostAlive(ip, opts.getPorts()) {
		return nil
	}

	now := time.Now()
	device := &model.DiscoveredDevice{
		ID:        uuid.Must(uuid.NewV7()).String(),
		IP:        ip,
		NetworkID: networkID,
		Status:    "online",
		FirstSeen: now,
		LastSeen:  now,
		Services:  []model.ServiceInfo{},
	}

	mac := s.arpScanner.LookupMAC(ip)
	if mac != "" {
		device.MACAddress = mac
	}

	scorer := NewConfidenceScorer()

	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		hostname := strings.TrimSuffix(names[0], ".")
		scorer.Add(hostname, "dns", GetHostnameSourceConfidence("dns"))
	}

	if opts.SSHCredID != "" {
		if sshResult, err := s.sshScanner.Scan(ctx, ip, opts.SSHCredID); err == nil {
			if sshResult.Hostname != "" {
				scorer.Add(sshResult.Hostname, "ssh", GetHostnameSourceConfidence("ssh"))
			}
			device.Services = append(device.Services, model.ServiceInfo{Port: 22, Protocol: "tcp", Service: "ssh"})

			if sshResult.OS != "" {
				device.OSGuess = sshResult.OS
			}
		}
	}

	if opts.SNMPCredID != "" {
		if snmpResult, err := s.snmpScanner.Scan(ctx, ip, opts.SNMPCredID); err == nil {
			if snmpResult.SysName != "" {
				scorer.Add(snmpResult.SysName, "snmp", GetHostnameSourceConfidence("snmp"))
			}
			device.Services = append(device.Services, model.ServiceInfo{Port: 161, Protocol: "udp", Service: "snmp"})

			if device.MACAddress == "" {
				for _, iface := range snmpResult.Interfaces {
					if iface.MAC != "" && iface.MAC != "00:00:00:00:00:00" {
						device.MACAddress = iface.MAC
						break
					}
				}
			}

			if device.MACAddress == "" {
				for _, entry := range snmpResult.ARPEntries {
					if entry.MAC != "" && entry.MAC != "00:00:00:00:00:00" {
						device.MACAddress = entry.MAC
						break
					}
				}
			}
		}
	}

	bestHostname, confidence := scorer.GetBest()
	if bestHostname != "" {
		device.Hostname = bestHostname
		device.Confidence = confidence
	}

	ports := s.scanPorts(ip, opts.getPorts())
	device.OpenPorts = ports

	banners := s.bannerGrabber.GrabBanners(ip, ports)
	for _, banner := range banners {
		device.Services = append(device.Services, model.ServiceInfo{
			Port:     banner.Port,
			Protocol: banner.Protocol,
			Service:  banner.Service,
			Version:  banner.Version,
		})
	}

	// OS fingerprinting (optional, for deep scans)
	if opts.ScanType == model.ScanTypeDeep && device.OSGuess == "" {
		fp := s.osFingerprinter.Fingerprint(ip)
		if fp.OSFamily != OSTypeUnknown {
			device.OSGuess = GetOSTypeFromFamily(fp.OSFamily)
			if fp.Confidence > device.Confidence {
				device.Confidence = fp.Confidence
			}
		}
	}

	// Vendor lookup from MAC address
	if device.MACAddress != "" && device.Vendor == "" {
		device.Vendor = s.ouiDatabase.Lookup(device.MACAddress)
	}

	return device
}

func (s *UnifiedScanner) isHostAlive(ip string, ports []int) bool {
	if len(ports) == 0 {
		ports = []int{22, 80, 443, 3389}
	}
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 2*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

func (s *UnifiedScanner) scanPorts(ip string, ports []int) []int {
	if len(ports) == 0 {
		ports = []int{22, 80, 443, 3389}
	}
	var open []int
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Second)
		if err == nil {
			conn.Close()
			open = append(open, port)
		}
	}
	return open
}

func (s *UnifiedScanner) cleanupCompletedScans() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, scan := range s.scans {
		if (scan.Status == model.ScanStatusCompleted || scan.Status == model.ScanStatusFailed) &&
			scan.CompletedAt != nil && time.Since(*scan.CompletedAt) > time.Hour {
			delete(s.scans, id)
		}
	}
}

func (opts *ScanOptions) getPorts() []int {
	if opts.Profile != nil && len(opts.Profile.Ports) > 0 {
		return opts.Profile.Ports
	}

	switch opts.ScanType {
	case model.ScanTypeQuick:
		return []int{22, 80, 443, 3389}
	case model.ScanTypeFull:
		return []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 8080}
	case model.ScanTypeDeep:
		return getTop100Ports()
	default:
		return []int{22, 80, 443, 3389}
	}
}
