package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type oauthReplayFixture struct {
	store   *storage.SQLiteStorage
	user    string
	svc     *OAuthService
	client  string
	refresh func(token string) (*model.OAuthTokenResponse, error)
	issue   func() string // a fresh refresh token for the user
}

func newOAuthReplayFixture(t *testing.T) *oauthReplayFixture {
	t.Helper()
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := SystemContext(context.Background(), "test")
	if err := store.CreateInitialAdmin(ctx, "root", "root@example.test", "Root", "password123"); err != nil {
		t.Fatal(err)
	}
	root, err := store.GetUserByUsername(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	client := &model.OAuthClient{Name: "agent", RedirectURIs: []string{"https://client.example/cb"}, Scope: "devices:read", CreatedAt: time.Now()}
	if err := store.CreateOAuthClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	svc := NewOAuthService(store, nil, "https://rackd.example.test")
	return &oauthReplayFixture{
		store:  store,
		user:   root.ID,
		svc:    svc,
		client: client.ID,
		refresh: func(token string) (*model.OAuthTokenResponse, error) {
			return svc.RefreshAccessToken(context.Background(), &model.OAuthTokenRequest{GrantType: "refresh_token", RefreshToken: token, ClientID: client.ID})
		},
		issue: func() string {
			plain, hash, err := auth.GenerateOAuthToken()
			if err != nil {
				t.Fatal(err)
			}
			if err := store.CreateOAuthToken(ctx, &model.OAuthToken{TokenType: "refresh", TokenHash: hash, ClientID: client.ID, UserID: root.ID, Scope: "devices:read", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
			return plain
		},
	}
}

// A thief who refreshes a stolen token first gets a new pair; when the real
// client then presents the old token, the thief's new tokens must die too.
func TestRefreshReplayRevokesTheThiefsTokens(t *testing.T) {
	f := newOAuthReplayFixture(t)
	stolen := f.issue()

	thief, err := f.refresh(stolen)
	if err != nil {
		t.Fatal(err)
	}
	// The thief rotates once more, so their token is several generations down.
	thief, err = f.refresh(thief.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.refresh(stolen); !errors.Is(err, storage.ErrOAuthTokenRevoked) {
		t.Fatalf("replaying the stolen token: %v, want revoked", err)
	}
	if _, err := f.refresh(thief.RefreshToken); err == nil {
		t.Error("the thief's latest refresh token still works after the replay")
	}
	if _, err := f.svc.ValidateAccessToken(thief.AccessToken); err == nil {
		t.Error("the thief's latest access token still works after the replay")
	}
}

// Two requests refreshing the same token at once: only one may get tokens.
func TestConcurrentRefreshesIssueOnePair(t *testing.T) {
	f := newOAuthReplayFixture(t)
	token := f.issue()

	var wg sync.WaitGroup
	results := make([]error, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, results[i] = f.refresh(token)
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, err := range results {
		if err == nil {
			ok++
		}
	}
	if ok > 1 {
		t.Errorf("%d concurrent refreshes succeeded, want at most 1", ok)
	}
}

// A password change ends OAuth access too, so a token granted from a
// compromised account doesn't outlive the reset.
func TestPasswordResetRevokesOAuthTokens(t *testing.T) {
	f := newOAuthReplayFixture(t)
	refresh := f.issue()
	pair, err := f.refresh(refresh)
	if err != nil {
		t.Fatal(err)
	}
	users := NewUserService(f.store, nil)
	if err := users.ResetPassword(SystemContext(context.Background(), "test"), f.user, &model.ResetPasswordRequest{NewPassword: "a-new-password"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.ValidateAccessToken(pair.AccessToken); err == nil {
		t.Error("the access token survived a password reset")
	}
	if _, err := f.refresh(pair.RefreshToken); err == nil {
		t.Error("the refresh token survived a password reset")
	}
}
