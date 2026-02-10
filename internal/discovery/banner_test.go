package discovery

import (
	"testing"
	"time"
)

func TestNewBannerGrabber(t *testing.T) {
	grabber := NewBannerGrabber(2 * time.Second)
	if grabber == nil {
		t.Fatal("NewBannerGrabber returned nil")
	}
	if grabber.timeout != 2*time.Second {
		t.Errorf("Expected timeout 2s, got %v", grabber.timeout)
	}
}

func TestBannerGrabber_GrabBanner(t *testing.T) {
	grabber := NewBannerGrabber(500 * time.Millisecond)

	// Test with unreachable port - should return nil
	banner := grabber.GrabBanner("127.0.0.1", 99999)
	if banner != nil {
		t.Errorf("Expected nil for unreachable port, got %+v", banner)
	}
}

func TestBannerGrabber_GrabBanners(t *testing.T) {
	grabber := NewBannerGrabber(500 * time.Millisecond)

	// Test with empty port list
	banners := grabber.GrabBanners("127.0.0.1", []int{})
	if len(banners) != 0 {
		t.Errorf("Expected empty banners for empty port list, got %d", len(banners))
	}

	// Test with unreachable ports
	banners = grabber.GrabBanners("127.0.0.1", []int{99999, 99998})
	if len(banners) != 0 {
		t.Errorf("Expected empty banners for unreachable ports, got %d", len(banners))
	}
}

func TestServiceBanner_Struct(t *testing.T) {
	banner := &ServiceBanner{
		Port:     80,
		Protocol: "tcp",
		Service:  "http",
		Version:  "nginx/1.18.0",
		Raw:      "HTTP/1.1 200 OK\r\nServer: nginx/1.18.0\r\n\r\n",
	}

	if banner.Port != 80 {
		t.Errorf("Expected port 80, got %d", banner.Port)
	}
	if banner.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", banner.Protocol)
	}
	if banner.Service != "http" {
		t.Errorf("Expected service http, got %s", banner.Service)
	}
	if banner.Version != "nginx/1.18.0" {
		t.Errorf("Expected version nginx/1.18.0, got %s", banner.Version)
	}
}

func TestBannerGrabber_Timeout(t *testing.T) {
	// Test short timeout
	grabber := NewBannerGrabber(1 * time.Millisecond)
	start := time.Now()
	banner := grabber.GrabBanner("192.168.255.255", 99999)
	duration := time.Since(start)

	if banner != nil {
		t.Errorf("Expected nil for timed out connection, got %+v", banner)
	}

	// Should complete quickly due to short timeout
	if duration > 100*time.Millisecond {
		t.Errorf("Expected quick timeout, got %v", duration)
	}
}
