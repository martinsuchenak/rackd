package discovery

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

type ARPEntry struct {
	IP  string
	MAC string
}

// ARPScanner caches the host's ARP table. It is shared by concurrent scans and
// a background load, so the table is replaced and read under a lock.
type ARPScanner struct {
	mu      sync.RWMutex
	entries []ARPEntry
}

func NewARPScanner() *ARPScanner {
	return &ARPScanner{entries: []ARPEntry{}}
}

// LoadARPTable reads the host's ARP table and replaces the cached entries.
func (s *ARPScanner) LoadARPTable() error {
	var entries []ARPEntry
	var err error
	switch runtime.GOOS {
	case "linux":
		entries, err = loadLinuxARP()
	case "darwin":
		entries, err = loadDarwinARP()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	if entries == nil {
		entries = []ARPEntry{}
	}
	s.mu.Lock()
	s.entries = entries
	s.mu.Unlock()
	return err
}

// Refresh reloads the ARP table.
func (s *ARPScanner) Refresh() error {
	return s.LoadARPTable()
}

func (s *ARPScanner) LookupMAC(ip string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, entry := range s.entries {
		if entry.IP == ip {
			return entry.MAC
		}
	}
	return ""
}

func loadLinuxARP() ([]ARPEntry, error) {
	file, err := os.Open("/proc/net/arp")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var entries []ARPEntry
	scanner := bufio.NewScanner(file)

	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 6 {
			ip := fields[0]
			mac := fields[3]
			if mac != "00:00:00:00:00:00" {
				if net.ParseIP(ip) != nil {
					if _, err := net.ParseMAC(mac); err == nil {
						entries = append(entries, ARPEntry{IP: ip, MAC: mac})
					}
				}
			}
		}
	}

	return entries, scanner.Err()
}

func loadDarwinARP() ([]ARPEntry, error) {
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var entries []ARPEntry
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		var ip, mac string
		if strings.Contains(line, "(") && strings.Contains(line, ")") {
			for i := 0; i < len(fields); i++ {
				if strings.HasPrefix(fields[i], "(") && strings.HasSuffix(fields[i], ")") {
					ip = strings.Trim(fields[i], "()")
					if i > 0 {
						mac = fields[i-1]
					}
					break
				}
			}
		} else {
			if len(fields) >= 4 {
				ip = strings.Trim(fields[1], "()")
				mac = fields[3]
			}
		}

		if ip != "" && mac != "" && mac != "00:00:00:00:00:00" && mac != "(incomplete)" {
			if net.ParseIP(ip) != nil {
				if _, err := net.ParseMAC(mac); err == nil {
					entries = append(entries, ARPEntry{IP: ip, MAC: mac})
				}
			}
		}
	}

	return entries, nil
}
