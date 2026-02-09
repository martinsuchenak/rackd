package discovery

import (
	"fmt"
	"net"
	"time"
)

const (
	OSTypeUnknown = "unknown"
	OSTypeLinux   = "linux"
	OSTypeWindows = "windows"
	OSTypeMacOS   = "macos"
	OSTypeNetwork = "network"
)

type OSFingerprint struct {
	TTL        uint8
	WindowSize uint16
	TCPFlags   string
	OSFamily   string
	Confidence int
}

type OSFingerprinter struct {
	timeout time.Duration
}

func NewOSFingerprinter(timeout time.Duration) *OSFingerprinter {
	return &OSFingerprinter{timeout: timeout}
}

func (f *OSFingerprinter) Fingerprint(ip string) *OSFingerprint {
	fp := &OSFingerprint{
		OSFamily:   OSTypeUnknown,
		Confidence: ConfidenceLow,
	}

	ttl := f.measureTTL(ip)
	fp.TTL = ttl

	windowSize := f.measureWindowSize(ip)
	fp.WindowSize = windowSize

	fp.OSFamily = f.classifyOS(ttl, windowSize)
	fp.Confidence = f.calculateConfidence(ttl, windowSize)

	return fp
}

func (f *OSFingerprinter) measureTTL(ip string) uint8 {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:80", ip), f.timeout)
	if err != nil {
		return 0
	}
	defer conn.Close()

	var buf [1]byte
	conn.SetReadDeadline(time.Now().Add(f.timeout))
	conn.SetWriteDeadline(time.Now().Add(f.timeout))

	n, err := conn.Read(buf[:])
	if err != nil || n == 0 {
		return 0
	}

	localAddr := conn.LocalAddr().(*net.TCPAddr)
	_, ipNet, _ := net.ParseCIDR(localAddr.String() + "/32")
	return ipNet.IP[8]
}

func (f *OSFingerprinter) measureTTLFromICMP(ip string) uint8 {
	conn, err := net.DialTimeout("ip4:icmp", ip, f.timeout)
	if err != nil {
		return 0
	}
	defer conn.Close()

	return 0
}

func (f *OSFingerprinter) measureWindowSize(ip string) uint16 {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", ip), f.timeout)
	if err != nil {
		return 0
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(f.timeout))
	conn.SetWriteDeadline(time.Now().Add(f.timeout))

	buf := make([]byte, 32)
	n, err := conn.Read(buf)
	if err != nil || n < 20 {
		return 0
	}

	if n >= 20 {
		return uint16(buf[18])<<8 | uint16(buf[19])
	}

	return 0
}

func (f *OSFingerprinter) classifyOS(ttl uint8, windowSize uint16) string {
	if ttl == 0 && windowSize == 0 {
		return OSTypeUnknown
	}

	switch ttl {
	case 64:
		if windowSize >= 5800 && windowSize <= 65535 {
			return OSTypeLinux
		}
		if windowSize >= 8192 && windowSize <= 65535 {
			return OSTypeMacOS
		}
		return OSTypeLinux
	case 128:
		if windowSize >= 8192 && windowSize <= 65535 {
			return OSTypeWindows
		}
		if windowSize >= 5792 && windowSize <= 65535 {
			return OSTypeLinux
		}
		return OSTypeWindows
	case 255:
		return OSTypeNetwork
	default:
		if windowSize >= 8192 {
			return OSTypeWindows
		}
		if windowSize >= 5800 {
			return OSTypeLinux
		}
		return OSTypeUnknown
	}
}

func (f *OSFingerprinter) calculateConfidence(ttl uint8, windowSize uint16) int {
	if ttl == 0 || windowSize == 0 {
		return ConfidenceLow
	}

	confidence := 0

	switch ttl {
	case 64:
		confidence += 2
	case 128:
		confidence += 2
	case 255:
		confidence += 2
	default:
		confidence += 1
	}

	if windowSize >= 8192 {
		confidence += 1
	} else if windowSize >= 5800 {
		confidence += 1
	}

	if confidence >= 4 {
		return ConfidenceHigh
	} else if confidence >= 3 {
		return ConfidenceMedium
	} else {
		return ConfidenceLow
	}
}

func (f *OSFingerprinter) GetOSFamily(ttl uint8, windowSize uint16) string {
	return f.classifyOS(ttl, windowSize)
}

func GetOSTypeFromFamily(family string) string {
	switch family {
	case OSTypeLinux:
		return "Linux"
	case OSTypeWindows:
		return "Windows"
	case OSTypeMacOS:
		return "macOS"
	case OSTypeNetwork:
		return "Network Device"
	default:
		return "Unknown"
	}
}
