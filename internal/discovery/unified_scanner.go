package discovery

import (
	"context"
	"fmt"
	"log"
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
	netbiosScanner  *NetBIOSScanner
	mdnsScanner     *mDNSScanner
	lldpScanner     *LLDPScanner
	adaptiveScanner *AdaptiveScanner
	correlator      *HostnameCorrelator
	classifier      *DeviceTypeClassifier
	mu              sync.RWMutex
}

func NewUnifiedScanner(store storage.DiscoveryStorage, netStore storage.NetworkStorage, credStore credentials.Storage, timeout time.Duration, snmpV2cEnabled bool) *UnifiedScanner {
	arpScanner := NewARPScanner()
	// Load ARP table asynchronously to avoid blocking server startup
	go arpScanner.LoadARPTable()

	return &UnifiedScanner{
		storage:         store,
		netStorage:      netStore,
		credStore:       credStore,
		scans:           make(map[string]*model.DiscoveryScan),
		cancelFuncs:     make(map[string]context.CancelFunc),
		arpScanner:      arpScanner,
		snmpScanner:     NewSNMPScanner(credStore, timeout, snmpV2cEnabled),
		sshScanner:      NewSSHScanner(credStore, timeout),
		bannerGrabber:   NewBannerGrabber(2 * time.Second),
		osFingerprinter: NewOSFingerprinter(2 * time.Second),
		ouiDatabase:     NewOUIDatabase(),
		netbiosScanner:  NewNetBIOSScanner(5 * time.Second),
		mdnsScanner:     NewmDNSScanner(5 * time.Second),
		lldpScanner:     NewLLDPScanner(5 * time.Second),
		adaptiveScanner: NewAdaptiveScanner(timeout, 10),
		correlator:      NewHostnameCorrelator(),
		classifier:      NewDeviceTypeClassifier(),
	}
}

func (s *UnifiedScanner) GetNetwork(ctx context.Context, id string) (*model.Network, error) {
	return s.netStorage.GetNetwork(ctx, id)
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

	// Validate subnet size
	ones, bits := ipNet.Mask.Size()
	if bits-ones > MaxSubnetBits {
		return nil, ErrSubnetTooLarge
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

	// Use a detached context for the background scan goroutine so it outlives
	// the HTTP request. Cancellation is handled explicitly via CancelScan().
	ctxCancellable, cancel := context.WithCancel(context.Background())

	s.mu.Lock()
	s.scans[scan.ID] = scan
	s.cancelFuncs[scan.ID] = cancel
	s.mu.Unlock()

	go s.runScanWithOptions(ctxCancellable, scan, network, ipNet, opts)

	return scan, nil
}

func (s *UnifiedScanner) GetScanStatus(ctx context.Context, scanID string) (*model.DiscoveryScan, error) {
	s.mu.RLock()
	scan, ok := s.scans[scanID]
	s.mu.RUnlock()
	if ok {
		return scan, nil
	}
	return s.storage.GetDiscoveryScan(ctx, scanID)
}

func (s *UnifiedScanner) CancelScan(ctx context.Context, scanID string) error {
	s.mu.Lock()
	scan, ok := s.scans[scanID]
	if !ok {
		s.mu.Unlock()
		return ErrScanNotFound
	}

	// Accept cancellation for pending or running scans
	if scan.Status != model.ScanStatusRunning && scan.Status != model.ScanStatusPending {
		s.mu.Unlock()
		return ErrScanNotRunning
	}

	cancel, ok := s.cancelFuncs[scanID]
	if !ok {
		s.mu.Unlock()
		return ErrScanNotFound
	}

	// Mark as failed with completed timestamp
	scan.Status = model.ScanStatusFailed
	scan.ErrorMessage = "scan cancelled"
	now := time.Now()
	scan.CompletedAt = &now

	s.mu.Unlock()

	// Cancel the context to stop running goroutines
	cancel()

	// Delete cancelFunc to prevent double-cancellation
	s.mu.Lock()
	delete(s.cancelFuncs, scanID)
	s.mu.Unlock()

	// Persist status to database
	if err := s.storage.UpdateDiscoveryScan(ctx, scan); err != nil {
		log.Printf("discovery: failed to update cancelled scan %s: %v", scanID, err)
	}

	return nil
}

// networkScanResults holds results from per-network broadcast scans run once before the per-host loop.
type networkScanResults struct {
	netbios map[string][]NetBIOSResult // keyed by IP
	mdns    map[string][]mDNSResult    // keyed by IP
	lldp    map[string]*LLDPResult     // keyed by mgmt IP
}

func (s *UnifiedScanner) runNetworkScans(ctx context.Context, subnet string, scanType string) *networkScanResults {
	results := &networkScanResults{
		netbios: make(map[string][]NetBIOSResult),
		mdns:    make(map[string][]mDNSResult),
		lldp:    make(map[string]*LLDPResult),
	}

	if scanType != model.ScanTypeFull && scanType != model.ScanTypeDeep {
		return results
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if nbResults, err := s.netbiosScanner.Discover(ctx, subnet); err == nil {
			for _, nb := range nbResults {
				results.netbios[nb.IP] = append(results.netbios[nb.IP], nb)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if mdnsResults, err := s.mdnsScanner.Discover(ctx, subnet); err == nil {
			for _, md := range mdnsResults {
				results.mdns[md.IP] = append(results.mdns[md.IP], md)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if lldpResults, err := s.lldpScanner.Discover(ctx); err == nil {
			for i := range lldpResults {
				if lldpResults[i].MgmtIP != "" {
					results.lldp[lldpResults[i].MgmtIP] = &lldpResults[i]
				}
			}
		}
	}()

	wg.Wait()
	return results
}

func (s *UnifiedScanner) runScanWithOptions(ctx context.Context, scan *model.DiscoveryScan, network *model.Network, ipNet *net.IPNet, opts *ScanOptions) {
	now := time.Now()
	scan.Status = model.ScanStatusRunning
	scan.StartedAt = &now
	if err := s.storage.UpdateDiscoveryScan(ctx, scan); err != nil {
		log.Printf("discovery: failed to update scan status: %v", err)
	}

	ips := expandCIDR(ipNet)
	scan.TotalHosts = len(ips)

	params := s.adaptiveScanner.CalculateParameters(network.Subnet, opts.ScanType)
	semaphore := make(chan struct{}, params.Workers)
	var wg sync.WaitGroup
	var foundCount int
	var scanMu sync.Mutex

	// Refresh ARP table before scanning to get recent MAC addresses
	s.arpScanner.Refresh()

	// Run per-network broadcast scans once (NetBIOS, mDNS, LLDP)
	netResults := s.runNetworkScans(ctx, network.Subnet, opts.ScanType)

	for i, ip := range ips {
		select {
		case <-ctx.Done():
			scan.Status = model.ScanStatusFailed
			scan.ErrorMessage = "scan cancelled"
			if err := s.storage.UpdateDiscoveryScan(ctx, scan); err != nil {
				log.Printf("discovery: failed to update cancelled scan: %v", err)
			}
			return
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(ip string, index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			device := s.discoverHostWithOptions(ctx, ip, network.ID, opts, params.Timeout, netResults)
			if device != nil {
				existing, _ := s.storage.GetDiscoveredDeviceByIP(ctx, network.ID, ip)
				if existing != nil {
					device.ID = existing.ID
					device.FirstSeen = existing.FirstSeen
					if err := s.storage.UpdateDiscoveredDevice(ctx, device); err != nil {
						log.Printf("discovery: failed to update device %s: %v", ip, err)
					}
				} else {
					if err := s.storage.CreateDiscoveredDevice(ctx, device); err != nil {
						log.Printf("discovery: failed to create device %s: %v", ip, err)
					}
				}

				scanMu.Lock()
				foundCount++
				scanMu.Unlock()
			}

			scanMu.Lock()
			scan.ScannedHosts = index + 1
			scan.FoundHosts = foundCount
			scan.ProgressPercent = float64(scan.ScannedHosts) / float64(scan.TotalHosts) * 100
			// Copy scan state under lock to avoid race with concurrent reads
			scanCopy := *scan
			scanMu.Unlock()

			_ = s.storage.UpdateDiscoveryScan(ctx, &scanCopy)
		}(ip, i)
	}

	wg.Wait()

	completedAt := time.Now()
	scan.Status = model.ScanStatusCompleted
	scan.CompletedAt = &completedAt
	if err := s.storage.UpdateDiscoveryScan(ctx, scan); err != nil {
		log.Printf("discovery: failed to update completed scan: %v", err)
	}

	s.cleanupCompletedScans()
}

func (s *UnifiedScanner) discoverHostWithOptions(ctx context.Context, ip string, networkID string, opts *ScanOptions, timeout time.Duration, netResults *networkScanResults) *model.DiscoveredDevice {
	// Check context at the very start
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	// Combined alive check + port scan: scan ports once and reuse results
	ports := s.scanPorts(ip, opts.getPorts(), timeout)
	if len(ports) == 0 {
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
		OpenPorts: ports,
		Services:  []model.ServiceInfo{},
	}

	mac := s.arpScanner.LookupMAC(ip)
	if mac != "" {
		device.MACAddress = mac
	}

	scorer := NewConfidenceScorer()

	// Check context before DNS lookup
	select {
	case <-ctx.Done():
		return nil
	default:
	}
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		hostname := strings.TrimSuffix(names[0], ".")
		scorer.Add(hostname, "dns", GetHostnameSourceConfidence("dns"))
	}

	if opts.SSHCredID != "" {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
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
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if snmpResult, err := s.snmpScanner.Scan(ctx, ip, opts.SNMPCredID); err == nil {
			if snmpResult.SysName != "" {
				scorer.Add(snmpResult.SysName, "snmp", GetHostnameSourceConfidence("snmp"))
			}
			device.Services = append(device.Services, model.ServiceInfo{Port: 161, Protocol: "udp", Service: "snmp"})

			if device.MACAddress == "" {
				for _, iface := range snmpResult.Interfaces {
					if iface.MAC != "" && iface.MAC != "00:00:00:00:00:00:00" {
						device.MACAddress = iface.MAC
						break
					}
				}
			}

			if device.MACAddress == "" {
				for _, entry := range snmpResult.ARPEntries {
					if entry.MAC != "" && entry.MAC != "00:00:00:00:00:00:00" {
						device.MACAddress = entry.MAC
						break
					}
				}
			}
		}
	}

	// Use pre-collected NetBIOS results (per-network scan already done)
	if netResults != nil {
		if nbResults, ok := netResults.netbios[ip]; ok {
			for _, nb := range nbResults {
				if nb.Hostname != "" && len(nb.Hostname) >= 3 && len(nb.Hostname) <= 15 {
					hasValidChars := true
					for _, c := range nb.Hostname {
						if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
							hasValidChars = false
							break
						}
					}
					if hasValidChars {
						scorer.Add(nb.Hostname, "netbios", GetHostnameSourceConfidence("netbios"))
						if device.OSGuess == "" {
							device.OSGuess = "Windows"
						}
					}
				}
			}
		}

		// Use pre-collected mDNS results
		if mdResults, ok := netResults.mdns[ip]; ok {
			for _, md := range mdResults {
				if md.Hostname != "" {
					scorer.Add(md.Hostname, "mdns", GetHostnameSourceConfidence("mdns"))
					if device.OSGuess == "" && strings.Contains(strings.ToLower(md.Type), "apple") {
						device.OSGuess = "macOS"
					}
				}
			}
		}

		// Use pre-collected LLDP results
		if lldpResult, ok := netResults.lldp[ip]; ok {
			if lldpResult.SystemName != "" {
				scorer.Add(lldpResult.SystemName, "lldp", GetHostnameSourceConfidence("snmp"))
			}
		}
	}

	// Use HostnameCorrelator for best hostname selection
	allSources := scorer.GetAll()
	if len(allSources) > 0 {
		conflict := s.correlator.Correlate(allSources)
		if conflict != nil && conflict.Recommended != "" {
			device.Hostname = conflict.Recommended
			_, confidence := scorer.GetBest()
			device.Confidence = confidence
		}
	}

	// Check context before banner grabbing
	select {
	case <-ctx.Done():
		return device
	default:
	}

	banners := s.bannerGrabber.GrabBanners(ip, ports)
	for _, banner := range banners {
		device.Services = append(device.Services, model.ServiceInfo{
			Port:     banner.Port,
			Protocol: banner.Protocol,
			Service:  banner.Service,
			Version:  banner.Version,
		})
	}

	// OS fingerprinting (for deep scans only, when no OS detected yet)
	if opts.ScanType == model.ScanTypeDeep && device.OSGuess == "" {
		select {
		case <-ctx.Done():
			return device
		default:
		}
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

	// Device type classification
	deviceInfo := &DeviceInfo{
		OS:     device.OSGuess,
		Vendor: device.Vendor,
		Ports:  device.OpenPorts,
	}
	for _, svc := range device.Services {
		deviceInfo.Services = append(deviceInfo.Services, ServiceInfo{
			Port:     svc.Port,
			Protocol: svc.Protocol,
			Service:  svc.Service,
			Version:  svc.Version,
		})
	}
	deviceType := s.classifier.Classify(deviceInfo)
	if deviceType != DeviceTypeUnknown {
		device.Status = "online:" + string(deviceType)
	}

	return device
}

func (s *UnifiedScanner) scanPorts(ip string, ports []int, timeout time.Duration) []int {
	if len(ports) == 0 {
		ports = []int{22, 80, 443, 3389}
	}
	var open []int
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
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
			scan.CompletedAt != nil && time.Since(*scan.CompletedAt) > 2*time.Minute {
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
