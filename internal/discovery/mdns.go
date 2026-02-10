package discovery

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"
)

type mDNSResult struct {
	Hostname string
	Type     string
	IP       string
}

type mDNSScanner struct {
	timeout time.Duration
}

func NewmDNSScanner(timeout time.Duration) *mDNSScanner {
	return &mDNSScanner{timeout: timeout}
}

func (s *mDNSScanner) Discover(ctx context.Context, network string) ([]mDNSResult, error) {
	if network == "" {
		return nil, fmt.Errorf("empty network")
	}

	_, _, err := net.ParseCIDR(network)
	if err != nil {
		return nil, fmt.Errorf("invalid network: %w", err)
	}

	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	resultChan := make(chan mDNSResult, 50)
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		buf := make([]byte, 1500)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				n, addr, err := conn.ReadFrom(buf)
				if err != nil {
					continue
				}

				results := s.parsemDNSResponse(buf[:n], addr)
				for _, r := range results {
					resultChan <- r
				}
			}
		}
	}()

	groupAddr := &net.UDPAddr{IP: net.ParseIP("224.0.0.251"), Port: 5353}
	query := s.buildmDNSQuery("_services._dns-sd._udp.local")

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		for i := 0; i < 2; i++ {
			select {
			case <-ctx.Done():
				goto cleanup
			default:
				if c, err := net.DialUDP("udp4", &net.UDPAddr{IP: nil, Port: 0}, groupAddr); err == nil {
					c.Write(query)
					c.Close()
				}
				time.Sleep(1 * time.Second)
			}
		}
	}

cleanup:
	close(resultChan)

	uniqueResults := make(map[string]mDNSResult)
	for r := range resultChan {
		key := r.IP + ":" + r.Hostname
		uniqueResults[key] = r
	}

	var results []mDNSResult
	for _, r := range uniqueResults {
		results = append(results, r)
	}

	return results, nil
}

func (s *mDNSScanner) buildmDNSQuery(name string) []byte {
	buf := make([]byte, 12)
	buf[0] = 0x00
	buf[1] = 0x00
	buf[2] = 0x00
	buf[3] = 0x00
	buf[4] = 0x00
	buf[5] = 0x01
	buf[6] = 0x00
	buf[7] = 0x00
	buf[8] = 0x00
	buf[9] = 0x00
	buf[10] = 0x00
	buf[11] = 0x00

	labels := strings.Split(name, ".")
	for _, label := range labels {
		buf = append(buf, byte(len(label)))
		buf = append(buf, []byte(label)...)
	}
	buf = append(buf, 0x00)

	buf = append(buf, 0x00, 0x0C, 0x00, 0x01)

	return buf
}

func (s *mDNSScanner) parsemDNSResponse(data []byte, addr net.Addr) []mDNSResult {
	if len(data) < 12 {
		return nil
	}

	flags := binary.BigEndian.Uint16(data[2:4])
	if flags&0x8000 == 0 {
		return nil
	}

	var results []mDNSResult
	offset := 12

	questions := binary.BigEndian.Uint16(data[4:6])
	answers := binary.BigEndian.Uint16(data[6:8])

	for i := uint16(0); i < questions; i++ {
		_, newOffset := s.parseName(data, offset)
		if newOffset == offset {
			break
		}
		offset = newOffset + 4
	}

	for i := uint16(0); i < answers; i++ {
		name, newOffset := s.parseName(data, offset)
		if newOffset == offset {
			break
		}
		offset = newOffset

		if offset+10 > len(data) {
			break
		}

		rrType := binary.BigEndian.Uint16(data[offset : offset+2])
		_ = binary.BigEndian.Uint32(data[offset+4 : offset+8])
		rdLen := binary.BigEndian.Uint16(data[offset+8 : offset+10])
		offset += 10

		if offset+int(rdLen) > len(data) {
			break
		}

		switch rrType {
		case 0x0001: // A record
			if int(rdLen) == 4 {
				ip := net.IP(data[offset : offset+4]).String()
				hostname := s.extractHostname(name)
				if hostname != "" {
					serviceType := s.getServiceType(name)
					results = append(results, mDNSResult{
						Hostname: hostname,
						Type:     serviceType,
						IP:       ip,
					})
				}
			}
		case 0x000C: // PTR record
			target, _ := s.parseName(data, offset)
			hostname := s.extractHostname(target)
			if hostname != "" {
				ip := s.extractIPFromAddr(addr)
				serviceType := s.getServiceType(name)
				results = append(results, mDNSResult{
					Hostname: hostname,
					Type:     serviceType,
					IP:       ip,
				})
			}
		}

		offset += int(rdLen)
	}

	return results
}

func (s *mDNSScanner) parseName(data []byte, offset int) (string, int) {
	if offset >= len(data) {
		return "", offset
	}

	var name strings.Builder
	// finalOffset tracks where the caller should continue reading.
	// Set on the first pointer jump and not updated after.
	finalOffset := -1
	currentOffset := offset
	jumps := 0

	for {
		if jumps > 5 {
			break
		}

		if currentOffset >= len(data) {
			break
		}

		labelLen := int(data[currentOffset])

		if labelLen == 0 {
			currentOffset++
			break
		}

		if labelLen&0xC0 == 0xC0 {
			if currentOffset+1 >= len(data) {
				break
			}
			if finalOffset < 0 {
				finalOffset = currentOffset + 2
			}
			pointer := int(binary.BigEndian.Uint16(data[currentOffset:currentOffset+2]) & 0x3FFF)
			jumps++
			currentOffset = pointer
			continue
		}

		currentOffset++
		if currentOffset+labelLen > len(data) {
			break
		}

		if name.Len() > 0 {
			name.WriteByte('.')
		}
		name.Write(data[currentOffset : currentOffset+labelLen])
		currentOffset += labelLen
	}

	if finalOffset >= 0 {
		return name.String(), finalOffset
	}
	return name.String(), currentOffset
}

func (s *mDNSScanner) extractHostname(name string) string {
	name = strings.TrimSuffix(name, ".local.")
	name = strings.TrimSuffix(name, ".local")
	name = strings.TrimSuffix(name, "._tcp.")
	name = strings.TrimSuffix(name, "._tcp")
	name = strings.TrimSuffix(name, "._udp.")
	name = strings.TrimSuffix(name, "._udp")
	name = strings.TrimSpace(name)

	parts := strings.Split(name, ".")
	for _, part := range parts {
		if !strings.HasPrefix(part, "_") && part != "" {
			return part
		}
	}

	return ""
}

func (s *mDNSScanner) getServiceType(name string) string {
	lower := strings.ToLower(name)

	if strings.Contains(lower, "_airplay") || strings.Contains(lower, "_raop") {
		return "Apple TV/AirPlay"
	}
	if strings.Contains(lower, "_afpovertcp") {
		return "File Sharing (AFP)"
	}
	if strings.Contains(lower, "_smb") {
		return "File Sharing (SMB)"
	}
	if strings.Contains(lower, "_ssh") {
		return "SSH"
	}
	if strings.Contains(lower, "_http") || strings.Contains(lower, "_https") {
		return "Web Server"
	}
	if strings.Contains(lower, "_ippusb") {
		return "Printer (USB)"
	}
	if strings.Contains(lower, "_ipp") {
		return "Printer (IPP)"
	}
	if strings.Contains(lower, "_printer") {
		return "Printer"
	}
	if strings.Contains(lower, "_chromecast") {
		return "Chromecast"
	}
	if strings.Contains(lower, "_googlecast") {
		return "Google Cast"
	}
	if strings.Contains(lower, "_spotify-connect") {
		return "Spotify Connect"
	}
	if strings.Contains(lower, "_hap") || strings.Contains(lower, "_homekit") {
		return "HomeKit"
	}
	if strings.Contains(lower, "_device-info") {
		return "Apple Device"
	}
	if strings.Contains(lower, "_services._dns-sd") {
		return "Service Discovery"
	}
	return "Unknown"
}

func (s *mDNSScanner) extractIPFromAddr(addr net.Addr) string {
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		return udpAddr.IP.String()
	}
	return ""
}
