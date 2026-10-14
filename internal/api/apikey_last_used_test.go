package api

import (
	"testing"
	"time"
)

func TestShouldRecordKeyUse(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) *time.Time { t := now.Add(d); return &t }
	for _, tc := range []struct {
		name     string
		lastUsed *time.Time
		want     bool
	}{
		{"never used", nil, true},
		{"used just now", at(0), false},
		{"used 59 s ago", at(-59 * time.Second), false},
		{"used a minute ago", at(-time.Minute), true},
		{"used a day ago", at(-24 * time.Hour), true},
		{"clock moved back", at(time.Hour), true},
	} {
		if got := shouldRecordKeyUse(tc.lastUsed, now); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
