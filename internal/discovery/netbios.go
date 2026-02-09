package discovery

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"strings"
	"time"
)

type NetBIOSResult struct {
	Hostname string
	IP       string
}

type NetBIOSScanner struct {
	timeout time.Duration
}

func NewNetBIOSScanner(timeout time.Duration) *NetBIOSScanner {
	return &NetBIOSScanner{timeout: timeout}
}

func (s *NetBIOSScanner) Discover(ctx context.Context, network string) ([]NetBIOSResult, error) {
	_, ipNet, err := net.ParseCIDR(network)
	if err != nil {
		return nil, err
	}

	var results []NetBIOSResult

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipAddr, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}

			if ipAddr.To4() == nil {
				continue
			}

			if !ipNet.Contains(ipAddr) {
				continue
			}

			bcastAddr := s.getBroadcastAddr(ipAddr, ipNet)
			if bcastAddr == nil {
				continue
			}

			ctx, cancel := context.WithTimeout(ctx, s.timeout)
			defer cancel()

			devices, err := s.scanNetwork(ctx, bcastAddr, ipAddr)
			if err != nil {
				continue
			}

			results = append(results, devices...)
		}
	}

	return results, nil
}

func (s *NetBIOSScanner) scanNetwork(ctx context.Context, broadcast, localIP net.IP) ([]NetBIOSResult, error) {
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resultChan := make(chan NetBIOSResult, 10)
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		buf := make([]byte, 512)
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

				hostname := s.parseNBNSResponse(buf[:n])
				if hostname != "" {
					resultChan <- NetBIOSResult{
						Hostname: hostname,
						IP:       addr.(*net.UDPAddr).IP.String(),
					}
				}
			}
		}
	}()

	query := s.buildNBNSQuery()
	targetAddr := &net.UDPAddr{IP: broadcast, Port: 137}

	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			goto cleanup
		default:
			_, err = conn.WriteTo(query, targetAddr)
			if err != nil {
				continue
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

cleanup:
	close(resultChan)

	var results []NetBIOSResult
	for r := range resultChan {
		results = append(results, r)
	}

	return results, nil
}

func (s *NetBIOSScanner) buildNBNSQuery() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, uint16(0x0000))
	binary.Write(buf, binary.BigEndian, uint16(0x0100))
	binary.Write(buf, binary.BigEndian, uint16(0x0001))
	binary.Write(buf, binary.BigEndian, uint16(0x0000))
	binary.Write(buf, binary.BigEndian, uint16(0x0000))
	binary.Write(buf, binary.BigEndian, uint16(0x0000))

	name := "*"
	encodedName := encodeNetBIOSName(name)
	buf.Write(encodedName)

	binary.Write(buf, binary.BigEndian, uint8(0x00))
	binary.Write(buf, binary.BigEndian, uint8(0x00))
	binary.Write(buf, binary.BigEndian, uint16(0x0021))
	binary.Write(buf, binary.BigEndian, uint16(0x0001))
	binary.Write(buf, binary.BigEndian, uint16(0x0000))
	binary.Write(buf, binary.BigEndian, uint16(0x0A20))
	binary.Write(buf, binary.BigEndian, uint16(0x0000))

	return buf.Bytes()
}

func encodeNetBIOSName(name string) []byte {
	encoded := make([]byte, 32)
	for i := 0; i < len(name) && i < 15; i++ {
		c := byte(name[i])
		encoded[i*2] = ((c >> 4) & 0x0F) + 'A'
		encoded[i*2+1] = (c & 0x0F) + 'A'
	}
	return encoded
}

func (s *NetBIOSScanner) parseNBNSResponse(data []byte) string {
	if len(data) < 57 {
		return ""
	}

	flags := binary.BigEndian.Uint16(data[2:4])
	if flags&0x8000 == 0 {
		return ""
	}

	nameLen := int(data[12])
	if nameLen > 32 {
		nameLen = 32
	}

	encodedName := data[13 : 13+nameLen]
	name := decodeNetBIOSName(encodedName)

	name = strings.TrimSpace(name)
	name = strings.TrimRight(name, " \u0000")

	if name == "" || name == "*" {
		return ""
	}

	return name
}

func decodeNetBIOSName(encoded []byte) string {
	var decoded strings.Builder
	for i := 0; i+1 < len(encoded); i += 2 {
		high := (encoded[i] - 'A') << 4
		low := encoded[i+1] - 'A'
		c := high | low
		if c >= 32 && c <= 126 {
			decoded.WriteByte(c)
		}
	}
	return decoded.String()
}

func (s *NetBIOSScanner) getBroadcastAddr(ip net.IP, ipNet *net.IPNet) net.IP {
	if ip.To4() == nil {
		return nil
	}

	ip = ip.To4()
	mask := ipNet.Mask

	broadcast := make(net.IP, len(ip))
	for i := range ip {
		broadcast[i] = ip[i] | (^mask[i])
	}

	return broadcast
}
