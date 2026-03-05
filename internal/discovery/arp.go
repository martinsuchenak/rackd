package discovery

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
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

func (s *ARPScanner) Refresh() error {
	s.entries = []ARPEntry{}
	return s.LoadARPTable()
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
				if net.ParseIP(ip) != nil {
					if _, err := net.ParseMAC(mac); err == nil {
						s.entries = append(s.entries, ARPEntry{IP: ip, MAC: mac})
					}
				}
			}
		}
	}

	return scanner.Err()
}

func (s *ARPScanner) loadDarwinARP() error {
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

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
					s.entries = append(s.entries, ARPEntry{IP: ip, MAC: mac})
				}
			}
		}
	}

	return nil
}
