package discovery

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"
)

type BannerGrabber struct {
	timeout time.Duration
}

func NewBannerGrabber(timeout time.Duration) *BannerGrabber {
	return &BannerGrabber{timeout: timeout}
}

type ServiceBanner struct {
	Port     int
	Protocol string
	Service  string
	Version  string
	Raw      string
}

func (b *BannerGrabber) GrabBanner(ip string, port int) *ServiceBanner {
	banner := &ServiceBanner{Port: port, Protocol: "tcp"}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), b.timeout)
	if err != nil {
		return nil
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(b.timeout))

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		response, err = reader.ReadString('\n')
		if err != nil {
			return nil
		}
	}

	banner.Raw = strings.TrimSpace(response)
	banner.Service, banner.Version = b.parseBanner(port, banner.Raw)

	return banner
}

func (b *BannerGrabber) parseBanner(port int, raw string) (service, version string) {
	raw = strings.TrimSpace(raw)

	switch port {
	case 21:
		if strings.HasPrefix(raw, "220") {
			parts := strings.Fields(raw)
			if len(parts) >= 3 {
				return "ftp", strings.Join(parts[2:], " ")
			}
			return "ftp", parts[0]
		}
	case 22:
		if strings.HasPrefix(raw, "SSH-") {
			parts := strings.SplitN(raw, "-", 3)
			if len(parts) >= 2 {
				return "ssh", parts[2]
			}
			return "ssh", ""
		}
	case 25:
		if strings.HasPrefix(raw, "220") {
			parts := strings.Fields(raw)
			if len(parts) >= 3 {
				return "smtp", strings.Join(parts[2:], " ")
			}
			return "smtp", parts[0]
		}
	case 80, 8080:
		if strings.Contains(raw, "Server:") {
			parts := strings.SplitN(raw, "Server:", 2)
			if len(parts) == 2 {
				return "http", strings.TrimSpace(parts[1])
			}
		}
		return "http", ""
	case 110:
		if strings.HasPrefix(raw, "+OK") {
			parts := strings.Fields(raw)
			if len(parts) >= 4 {
				return "pop3", strings.Join(parts[3:], " ")
			}
			return "pop3", ""
		}
	case 143:
		if strings.HasPrefix(raw, "* OK") {
			parts := strings.Fields(raw)
			if len(parts) >= 5 {
				return "imap", strings.Join(parts[4:], " ")
			}
			return "imap", ""
		}
	case 443:
		if len(raw) > 0 && bytes.Contains([]byte(raw), []byte{0x15, 0x03}) {
			return "https", "TLS/SSL"
		}
		if bytes.Contains([]byte(raw), []byte("HTTP/")) {
			return "https", "HTTP/1.1"
		}
	case 3306:
		if bytes.Contains([]byte(raw), []byte{0x0a}) {
			parts := bytes.Split([]byte(raw), []byte{0x0a})
			if len(parts) >= 2 {
				return "mysql", string(parts[1])
			}
			return "mysql", ""
		}
	case 3389:
		if len(raw) > 0 {
			return "rdp", "Microsoft Terminal Services"
		}
	case 5432:
		if len(raw) > 0 {
			return "postgresql", ""
		}
	}

	return "", ""
}

func (b *BannerGrabber) GrabBanners(ip string, ports []int) []*ServiceBanner {
	var banners []*ServiceBanner

	for _, port := range ports {
		if banner := b.GrabBanner(ip, port); banner != nil {
			banners = append(banners, banner)
		}
	}

	return banners
}
