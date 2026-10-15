package service

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// ErrTooManyLoginAttempts is returned when an account has had too many failed
// sign-ins recently.
var ErrTooManyLoginAttempts = errors.New("too many failed sign-ins for this account; try again later")

// The per-IP login limit doesn't stop guessing from many addresses (an IPv6
// /64 or a botnet), so failures are also counted per account.
const (
	defaultMaxLoginFailures   = 10
	defaultLoginFailureWindow = 15 * time.Minute
	maxThrottledAccounts      = 10000
)

// loginThrottle counts failed sign-ins per username. It is per process: with
// several Rackd instances each counts separately.
type loginThrottle struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	failures map[string]*loginFailures
	now      func() time.Time
}

type loginFailures struct {
	count int
	since time.Time
}

func newLoginThrottle(max int, window time.Duration) *loginThrottle {
	return &loginThrottle{max: max, window: window, failures: map[string]*loginFailures{}, now: time.Now}
}

func throttleKey(username string) string { return strings.ToLower(username) }

// blocked reports whether username has reached the failure limit.
func (t *loginThrottle) blocked(username string) bool {
	if t == nil || t.max <= 0 {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	f, ok := t.failures[throttleKey(username)]
	if !ok {
		return false
	}
	if t.now().Sub(f.since) > t.window {
		delete(t.failures, throttleKey(username))
		return false
	}
	return f.count >= t.max
}

// fail records a failed sign-in.
func (t *loginThrottle) fail(username string) {
	if t == nil || t.max <= 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	key := throttleKey(username)
	f, ok := t.failures[key]
	if !ok || now.Sub(f.since) > t.window {
		if len(t.failures) >= maxThrottledAccounts {
			t.prune(now)
		}
		if len(t.failures) >= maxThrottledAccounts {
			return // full of live entries; the per-IP limit still applies
		}
		f = &loginFailures{since: now}
		t.failures[key] = f
	}
	f.count++
}

// reset clears an account's failures after a successful sign-in.
func (t *loginThrottle) reset(username string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	delete(t.failures, throttleKey(username))
	t.mu.Unlock()
}

func (t *loginThrottle) prune(now time.Time) {
	for key, f := range t.failures {
		if now.Sub(f.since) > t.window {
			delete(t.failures, key)
		}
	}
}
