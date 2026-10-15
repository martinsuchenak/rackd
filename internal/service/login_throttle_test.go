package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/storage"
)

// Guessing from many addresses defeats a per-IP limit, so failures are also
// counted per account, and past the limit even the right password is refused
// until the window passes.
func TestLoginThrottlesAnAccountAfterRepeatedFailures(t *testing.T) {
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.CreateInitialAdmin(ctx, "root", "root@example.test", "Root", "password123"); err != nil {
		t.Fatal(err)
	}
	svc := NewAuthService(store, auth.NewSessionManager(time.Hour, nil))
	svc.SetLoginThrottle(3, 15*time.Minute)
	now := time.Now()
	svc.throttle.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if _, err := svc.Login(ctx, "Root", "wrong"); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("failure %d: %v", i+1, err)
		}
	}
	if _, err := svc.Login(ctx, "root", "password123"); !errors.Is(err, ErrTooManyLoginAttempts) {
		t.Errorf("right password after the limit: %v, want ErrTooManyLoginAttempts", err)
	}
	// Unknown usernames are throttled the same way, so the limit reveals nothing.
	for i := 0; i < 3; i++ {
		_, _ = svc.Login(ctx, "nobody", "x")
	}
	if _, err := svc.Login(ctx, "nobody", "x"); !errors.Is(err, ErrTooManyLoginAttempts) {
		t.Errorf("unknown user after the limit: %v", err)
	}

	now = now.Add(16 * time.Minute)
	if _, err := svc.Login(ctx, "root", "password123"); err != nil {
		t.Errorf("after the window: %v", err)
	}
	// A success clears the count.
	_, _ = svc.Login(ctx, "root", "wrong")
	_, _ = svc.Login(ctx, "root", "wrong")
	if _, err := svc.Login(ctx, "root", "password123"); err != nil {
		t.Errorf("below the limit: %v", err)
	}
}
