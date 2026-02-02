package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type AdvancedDiscoveryService struct {
	storage     storage.DiscoveryStorage
	netStorage  storage.NetworkStorage
	credStore   credentials.Storage
	snmpScanner *SNMPScanner
	sshScanner  *SSHScanner
	scans       map[string]*model.DiscoveryScan
	mu          sync.RWMutex
}

func NewAdvancedDiscoveryService(store storage.DiscoveryStorage, netStore storage.NetworkStorage, credStore credentials.Storage, timeout time.Duration) *AdvancedDiscoveryService {
	return &AdvancedDiscoveryService{
		storage:     store,
		netStorage:  netStore,
		credStore:   credStore,
		snmpScanner: NewSNMPScanner(credStore, timeout),
		sshScanner:  NewSSHScanner(credStore, timeout),
		scans:       make(map[string]*model.DiscoveryScan),
	}
}

func (s *AdvancedDiscoveryService) ScanAdvanced(ctx context.Context, network *model.Network, profile *model.ScanProfile, snmpCredID, sshCredID string) (*model.DiscoveryScan, error) {
	_, ipNet, err := net.ParseCIDR(network.Subnet)
	if err != nil {
		return nil, err
	}

	scan := &model.DiscoveryScan{
		ID:         uuid.Must(uuid.NewV7()).String(),
		NetworkID:  network.ID,
		Status:     model.ScanStatusPending,
		ScanType:   profile.ScanType,
		TotalHosts: countHosts(ipNet),
	}

	if err := s.storage.CreateDiscoveryScan(scan); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.scans[scan.ID] = scan
	s.mu.Unlock()

	go s.runScan(ctx, scan, network, ipNet, profile, snmpCredID, sshCredID)

	return scan, nil
}

func (s *AdvancedDiscoveryService) GetScanStatus(scanID string) (*model.DiscoveryScan, error) {
	s.mu.RLock()
	scan, ok := s.scans[scanID]
	s.mu.RUnlock()
	if ok {
		return scan, nil
	}
	return s.storage.GetDiscoveryScan(scanID)
}

func (s *AdvancedDiscoveryService) GetNetwork(id string) (*model.Network, error) {
	return s.netStorage.GetNetwork(id)
}

func (s *AdvancedDiscoveryService) runScan(ctx context.Context, scan *model.DiscoveryScan, network *model.Network, ipNet *net.IPNet, profile *model.ScanProfile, snmpCredID, sshCredID string) {
	now := time.Now()
	scan.Status = model.ScanStatusRunning
	scan.StartedAt = &now
	s.storage.UpdateDiscoveryScan(scan)

	ips := expandCIDR(ipNet)
	scan.TotalHosts = len(ips)

	semaphore := make(chan struct{}, profile.MaxWorkers)
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

			device := s.discoverHost(ctx, ip, network.ID, profile, snmpCredID, sshCredID)
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
}

func (s *AdvancedDiscoveryService) discoverHost(ctx context.Context, ip string, networkID string, profile *model.ScanProfile, snmpCredID, sshCredID string) *model.DiscoveredDevice {
	if !s.isHostAlive(ip, profile.Ports) {
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
		OpenPorts: s.scanPorts(ip, profile.Ports),
		Services:  []model.ServiceInfo{},
	}

	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		device.Hostname = names[0]
	}

	if profile.EnableSNMP && snmpCredID != "" {
		if snmpResult, err := s.snmpScanner.Scan(ctx, ip, snmpCredID); err == nil {
			if snmpResult.SysName != "" {
				device.Hostname = snmpResult.SysName
			}
			device.Services = append(device.Services, model.ServiceInfo{Port: 161, Protocol: "udp", Service: "snmp"})
		}
	}

	if profile.EnableSSH && sshCredID != "" {
		if sshResult, err := s.sshScanner.Scan(ctx, ip, sshCredID); err == nil {
			if sshResult.Hostname != "" && device.Hostname == "" {
				device.Hostname = sshResult.Hostname
			}
			device.Services = append(device.Services, model.ServiceInfo{Port: 22, Protocol: "tcp", Service: "ssh"})
		}
	}

	return device
}

func (s *AdvancedDiscoveryService) isHostAlive(ip string, ports []int) bool {
	if len(ports) == 0 {
		ports = []int{22, 80, 443, 3389}
	}
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, fmt.Sprintf("%d", port)), 2*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

func (s *AdvancedDiscoveryService) scanPorts(ip string, ports []int) []int {
	if len(ports) == 0 {
		ports = []int{22, 80, 443, 3389, 8080}
	}
	var open []int
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, fmt.Sprintf("%d", port)), time.Second)
		if err == nil {
			conn.Close()
			open = append(open, port)
		}
	}
	return open
}

// countHosts, expandCIDR, and incrementIP are defined in scanner.go
