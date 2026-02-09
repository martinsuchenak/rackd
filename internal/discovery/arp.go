package discovery

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
)

type ARPEntry struct {
	IP  string
	MAC string
}

type ARPScanner struct {
	entries []ARPEntry
}

func NewARPScanner() *ARPScanner {
	return &ARPScanner{entries: []ARPEntry{}}
}

func (s *ARPScanner) LoadARPTable() error {
	switch runtime.GOOS {
	case "linux":
		return s.loadLinuxARP()
	case "darwin":
		return s.loadDarwinARP()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (s *ARPScanner) LookupMAC(ip string) string {
	for _, entry := range s.entries {
		if entry.IP == ip {
			return entry.MAC
		}
	}
	return ""
}

func (s *ARPScanner) loadLinuxARP() error {
	file, err := os.Open("/proc/net/arp")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 6 {
			ip := fields[0]
			mac := fields[3]
			if mac != "00:00:00:00:00:00" {
				s.entries = append(s.entries, ARPEntry{IP: ip, MAC: mac})
			}
		}
	}

	return scanner.Err()
}

func (s *ARPScanner) loadDarwinARP() error {
	return fmt.Errorf("darwin ARP scanning not yet implemented")
}
