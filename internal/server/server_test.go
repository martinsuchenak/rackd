package server

import (
	"testing"
	"time"
)

func TestWriteTimeoutOutlastsRequestTimeout(t *testing.T) {
	tests := []struct {
		requestTimeout time.Duration
		want           time.Duration
	}{
		{0, 15 * time.Second},                // request timeout disabled: keep the default
		{5 * time.Second, 15 * time.Second},  // never below the default
		{30 * time.Second, 35 * time.Second}, // default REQUEST_TIMEOUT
		{2 * time.Minute, 2*time.Minute + 5*time.Second},
	}
	for _, tt := range tests {
		got := writeTimeout(tt.requestTimeout)
		if got != tt.want {
			t.Errorf("writeTimeout(%v) = %v, want %v", tt.requestTimeout, got, tt.want)
		}
		if tt.requestTimeout > 0 && got <= tt.requestTimeout {
			t.Errorf("writeTimeout(%v) = %v must exceed the request timeout", tt.requestTimeout, got)
		}
	}
}
