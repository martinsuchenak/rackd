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

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		if s.hasMulticast(iface) {
			multicastAddr := &net.UDPAddr{IP: net.ParseIP("01:80:c2:00:00:0e"), Port: 0}
			if c, err := net.DialUDP("udp4", &net.UDPAddr{IP: nil, Port: 0}, multicastAddr); err == nil {
				c.SetDeadline(time.Now().Add(s.timeout))
				c.Close()
			}
		}
	}

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

	for offset+4 <= len(data) {
		tlvType := (data[offset] >> 1) & 0x7F
		tlvLen := int(binary.BigEndian.Uint16(data[offset:offset+2]) & 0x01FF)
		offset += 2

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

	result.MgmtIP = s.extractIPFromAddr(addr)

	return result
}

func (s *LLDPScanner) parseChassisID(data []byte, result *LLDPResult) {
	if len(data) < 2 {
		return
	}

	chassisType := data[0]
	chassisID := string(data[1:])

	switch chassisType {
	case 1:
		result.ChassisType = "MAC"
	case 2:
		result.ChassisType = "Network Address"
	case 3:
		result.ChassisType = "Interface Name"
	case 4:
		result.ChassisType = "Interface Alias"
	case 5:
		result.ChassisType = "Chassis Component"
	case 6:
		result.ChassisType = "Port Component"
	case 7:
		result.ChassisType = "MAC Address"
	default:
		result.ChassisType = fmt.Sprintf("Type %d", chassisType)
	}

	result.ChassisID = chassisID
}

func (s *LLDPScanner) parsePortID(data []byte, result *LLDPResult) {
	if len(data) < 2 {
		return
	}

	portType := data[0]
	portID := string(data[1:])

	switch portType {
	case 1:
		result.PortID = portID
	case 2:
		result.PortID = portID
	case 3:
		result.PortID = portID
	case 4:
		result.PortID = portID
	case 5:
		result.PortID = portID
	case 6:
		result.PortID = portID
	case 7:
		result.PortID = portID
	default:
		result.PortID = portID
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
	if len(data) < 1 {
		return
	}

	addrFamily := data[0]
	if addrFamily == 1 && len(data) >= 5 {
		ip := net.IP(data[1:5]).String()
		if result.MgmtIP == "" {
			result.MgmtIP = ip
		}
	} else if addrFamily == 2 && len(data) >= 17 {
		ip := net.IP(data[1:17]).String()
		if result.MgmtIP == "" {
			result.MgmtIP = ip
		}
	}
}

func (s *LLDPScanner) hasMulticast(iface net.Interface) bool {
	addrs, err := iface.MulticastAddrs()
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.To4() != nil {
				return true
			}
		}
	}
	return false
}

func (s *LLDPScanner) extractIPFromAddr(addr net.Addr) string {
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		return udpAddr.IP.String()
	}
	return ""
}
