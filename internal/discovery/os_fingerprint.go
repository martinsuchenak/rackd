package discovery

import (
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
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

	// Validate IP
	if ip == "" || net.ParseIP(ip) == nil {
		return fp
	}

	ttl := f.measureTTL(ip)
	fp.TTL = ttl

	fp.OSFamily = f.classifyOS(ttl)
	fp.Confidence = f.calculateConfidence(ttl)

	return fp
}

// measureTTL uses ping to determine the TTL of a remote host.
func (f *OSFingerprinter) measureTTL(ip string) uint8 {
	// Defense-in-depth: validate IP address before passing to exec.Command
	// Although Fingerprint() validates before calling this, we validate here
	// to ensure safety if called from elsewhere in the future.
	if ip == "" || net.ParseIP(ip) == nil {
		return 0
	}

	timeoutSec := int(f.timeout.Seconds())
	if timeoutSec < 1 {
		timeoutSec = 1
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("ping", "-c", "1", "-t", strconv.Itoa(timeoutSec), ip)
	default: // linux and others
		cmd = exec.Command("ping", "-c", "1", "-W", strconv.Itoa(timeoutSec), ip)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0
	}

	return parseTTLFromPing(string(output))
}

var ttlRegex = regexp.MustCompile(`(?i)ttl=(\d+)`)

func parseTTLFromPing(output string) uint8 {
	matches := ttlRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0
	}

	ttl, err := strconv.Atoi(matches[1])
	if err != nil || ttl < 0 || ttl > 255 {
		return 0
	}

	return uint8(ttl)
}

func (f *OSFingerprinter) classifyOS(ttl uint8) string {
	if ttl == 0 {
		return OSTypeUnknown
	}

	// TTL ranges: Linux/macOS typically start at 64, Windows at 128, network devices at 255
	// Observed TTL may be lower due to hops, so we use ranges
	switch {
	case ttl <= 64:
		return OSTypeLinux // or macOS — both use initial TTL 64
	case ttl <= 128:
		return OSTypeWindows
	default:
		return OSTypeNetwork
	}
}

func (f *OSFingerprinter) calculateConfidence(ttl uint8) int {
	if ttl == 0 {
		return ConfidenceLow
	}

	// Exact initial TTL values give higher confidence
	switch ttl {
	case 64:
		return ConfidenceHigh
	case 128:
		return ConfidenceHigh
	case 255:
		return ConfidenceHigh
	default:
		return ConfidenceMedium
	}
}

func (f *OSFingerprinter) GetOSFamily(ttl uint8, windowSize uint16) string {
	return f.classifyOS(ttl)
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
