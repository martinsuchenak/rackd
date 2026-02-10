package discovery

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type LLDPResult struct {
	ChassisID   string
	ChassisType string
	PortID      string
	PortDesc    string
	SystemName  string
	SystemDesc  string
	MgmtIP      string
}

type LLDPScanner struct {
	timeout time.Duration
}

func NewLLDPScanner(timeout time.Duration) *LLDPScanner {
	return &LLDPScanner{timeout: timeout}
}

// Discover listens for LLDP frames. Note: LLDP operates at Layer 2 (Ethernet),
// so this implementation can only capture LLDP data that has been bridged to UDP
// or in environments where raw Ethernet frames are accessible via UDP.
func (s *LLDPScanner) Discover(ctx context.Context) ([]LLDPResult, error) {
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resultChan := make(chan LLDPResult, 10)
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

				result := s.parseLLDP(buf[:n], addr)
				if result != nil {
					resultChan <- *result
				}
			}
		}
	}()

	time.Sleep(s.timeout)
	close(resultChan)

	var results []LLDPResult
	for r := range resultChan {
		results = append(results, r)
	}

	return results, nil
}

func (s *LLDPScanner) parseLLDP(data []byte, addr net.Addr) *LLDPResult {
	if len(data) < 14 {
		return nil
	}

	ethDest := data[0:6]
	if !bytes.Equal(ethDest, []byte{0x01, 0x80, 0xc2, 0x00, 0x00, 0x0e}) &&
		!bytes.Equal(ethDest, []byte{0x01, 0x80, 0xc2, 0x00, 0x00, 0x03}) &&
		!bytes.Equal(ethDest, []byte{0x01, 0x80, 0xc2, 0x00, 0x00, 0x00}) {
		return nil
	}

	ethType := binary.BigEndian.Uint16(data[12:14])
	if ethType != 0x88cc {
		return nil
	}

	result := &LLDPResult{}
	offset := 14

	for offset+2 <= len(data) {
		tlvType := (data[offset] >> 1) & 0x7F
		tlvLen := int(binary.BigEndian.Uint16(data[offset:offset+2]) & 0x01FF)
		offset += 2

		if tlvLen == 0 && tlvType == 0 {
			break // End of LLDPDU
		}

		if offset+tlvLen > len(data) {
			break
		}

		tlvValue := data[offset : offset+tlvLen]

		switch tlvType {
		case 1:
			s.parseChassisID(tlvValue, result)
		case 2:
			s.parsePortID(tlvValue, result)
		case 4:
			s.parsePortDesc(tlvValue, result)
		case 5:
			s.parseSystemName(tlvValue, result)
		case 6:
			s.parseSystemDesc(tlvValue, result)
		case 8:
			s.parseMgmtIP(tlvValue, result)
		}

		offset += tlvLen
	}

	if result.ChassisID == "" && result.SystemName == "" {
		return nil
	}

	if result.MgmtIP == "" {
		result.MgmtIP = s.extractIPFromAddr(addr)
	}

	return result
}

func (s *LLDPScanner) parseChassisID(data []byte, result *LLDPResult) {
	if len(data) < 2 {
		return
	}

	chassisSubtype := data[0]
	payload := data[1:]

	switch chassisSubtype {
	case 4, 7: // MAC Address subtypes
		result.ChassisType = "MAC"
		if len(payload) >= 6 {
			result.ChassisID = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
				payload[0], payload[1], payload[2], payload[3], payload[4], payload[5])
		}
	case 5: // Network Address
		result.ChassisType = "Network Address"
		if len(payload) >= 5 && payload[0] == 1 { // IPv4
			result.ChassisID = net.IP(payload[1:5]).String()
		} else {
			result.ChassisID = string(payload)
		}
	default:
		result.ChassisType = fmt.Sprintf("Type %d", chassisSubtype)
		result.ChassisID = string(payload)
	}
}

func (s *LLDPScanner) parsePortID(data []byte, result *LLDPResult) {
	if len(data) < 2 {
		return
	}

	portSubtype := data[0]
	payload := data[1:]

	switch portSubtype {
	case 3: // MAC Address
		if len(payload) >= 6 {
			result.PortID = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
				payload[0], payload[1], payload[2], payload[3], payload[4], payload[5])
		}
	default: // Interface name, alias, etc.
		result.PortID = string(payload)
	}
}

func (s *LLDPScanner) parsePortDesc(data []byte, result *LLDPResult) {
	result.PortDesc = string(data)
}

func (s *LLDPScanner) parseSystemName(data []byte, result *LLDPResult) {
	result.SystemName = string(data)
}

func (s *LLDPScanner) parseSystemDesc(data []byte, result *LLDPResult) {
	result.SystemDesc = string(data)
}

func (s *LLDPScanner) parseMgmtIP(data []byte, result *LLDPResult) {
	if len(data) < 2 {
		return
	}

	addrLen := int(data[0])
	if addrLen < 2 || len(data) < 1+addrLen {
		return
	}

	addrSubtype := data[1]
	if addrSubtype == 1 && addrLen >= 5 { // IPv4
		ip := net.IP(data[2:6]).String()
		if result.MgmtIP == "" {
			result.MgmtIP = ip
		}
	} else if addrSubtype == 2 && addrLen >= 17 { // IPv6
		ip := net.IP(data[2:18]).String()
		if result.MgmtIP == "" {
			result.MgmtIP = ip
		}
	}
}

func (s *LLDPScanner) extractIPFromAddr(addr net.Addr) string {
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		return udpAddr.IP.String()
	}
	return ""
}
