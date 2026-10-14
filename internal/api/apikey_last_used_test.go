package api

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
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
		{"clock 50 ms behind another server", at(50 * time.Millisecond), false},
		{"clock moved back an hour", at(time.Hour), true},
	} {
		if got := shouldRecordKeyUse(tc.lastUsed, now); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// countingStore counts UpdateAPIKeyLastUsed calls and passes them through.
type countingStore struct {
	storage.ExtendedStorage
	updates atomic.Int32
}

func (c *countingStore) UpdateAPIKeyLastUsed(ctx context.Context, id string, at time.Time) error {
	c.updates.Add(1)
	return c.ExtendedStorage.UpdateAPIKeyLastUsed(ctx, id, at)
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatal("condition not met within 2s")
}

func TestAuthenticateAPIKeyRecordsUseOncePerInterval(t *testing.T) {
	_, base := setupTestHandler(t)
	defer func() { _ = base.Close() }()
	store := &countingStore{ExtendedStorage: base}
	ctx := context.Background()

	const token = "rk_test_last_used"
	key := &model.APIKey{Name: "last-used", Key: auth.HashToken(token), UserID: "test-user-id"}
	if err := base.CreateAPIKey(ctx, key); err != nil {
		t.Fatal(err)
	}

	// First use writes last_used_at.
	if _, err := AuthenticateAPIKey(ctx, store, token, "127.0.0.1", "test"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		k, err := base.GetAPIKeyByKey(ctx, key.Key)
		return err == nil && k.LastUsedAt != nil
	})

	// Further use within the interval does not.
	for range 5 {
		if _, err := AuthenticateAPIKey(ctx, store, token, "127.0.0.1", "test"); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(50 * time.Millisecond)
	if n := store.updates.Load(); n != 1 {
		t.Errorf("UpdateAPIKeyLastUsed called %d times, want 1", n)
	}

	// A rejected request (owner deactivated) does not count as use.
	if err := base.UpdateAPIKeyLastUsed(ctx, key.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	user, err := base.GetUser(ctx, "test-user-id")
	if err != nil {
		t.Fatal(err)
	}
	user.IsActive = false
	if err := base.UpdateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateAPIKey(ctx, store, token, "127.0.0.1", "test"); err == nil {
		t.Fatal("key of an inactive user authenticated")
	}
	time.Sleep(50 * time.Millisecond)
	if n := store.updates.Load(); n != 1 {
		t.Errorf("rejected request updated last_used_at (%d updates)", n)
	}
}
