package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type extendedTestHandler struct {
	handler    *Handler
	store      storage.ExtendedStorage
	credStore  credentials.Storage
	profiles   storage.ProfileStorage
	scheduled  storage.ScheduledScanStorage
	mux        *http.ServeMux
}

func setupExtendedTestHandler(t *testing.T, withSessions, withCredentials, withProfiles, withScheduled bool) *extendedTestHandler {
	t.Helper()

	baseHandler, store := setupTestHandler(t)
	services := service.NewServices(store, nil, nil)

	opts := []HandlerOption{
		WithServices(services),
	}

	result := &extendedTestHandler{store: store}

	if withSessions {
		sessionManager := auth.NewSessionManager(time.Hour, nil)
		opts = append(opts, WithSessionManager(sessionManager))
		services = service.NewServices(store, sessionManager, nil)
		opts[0] = WithServices(services)
	}

	if withCredentials {
		credStore, err := credentials.NewSQLiteStorage(store.DB(), []byte("0123456789abcdef0123456789abcdef"))
		if err != nil {
			t.Fatalf("failed to create credential storage: %v", err)
		}
		result.credStore = credStore
		services.SetCredentialsStorage(credStore)
		opts = append(opts, WithCredentialsStorage(credStore))
	}

	if withProfiles {
		profileStore, err := storage.NewSQLiteProfileStorage(store.DB())
		if err != nil {
			t.Fatalf("failed to create profile storage: %v", err)
		}
		result.profiles = profileStore
		services.SetProfileStorage(profileStore)
		opts = append(opts, WithProfileStorage(profileStore))
	}

	if withScheduled {
		scheduledStore, err := storage.NewSQLiteScheduledScanStorage(store.DB())
		if err != nil {
			t.Fatalf("failed to create scheduled scan storage: %v", err)
		}
		result.scheduled = scheduledStore
		services.SetScheduledScanStorage(scheduledStore)
		opts = append(opts, WithScheduledScanStorage(scheduledStore))
	}

	h := NewHandler(store, baseHandler.scanner, opts...)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	result.handler = h
	result.mux = mux
	return result
}

func (e *extendedTestHandler) close() {
	if e.store != nil {
		_ = e.store.Close()
	}
}

func setupDNSTestHandler(t *testing.T) *extendedTestHandler {
	t.Helper()

	baseHandler, store := setupTestHandler(t)
	services := service.NewServices(store, nil, nil)

	encryptor, err := credentials.NewEncryptor([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("failed to create DNS encryptor: %v", err)
	}
	services.SetDNSService(store, encryptor)

	h := NewHandler(store, baseHandler.scanner, WithServices(services))
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	return &extendedTestHandler{
		handler: h,
		store:   store,
		mux:     mux,
	}
}

func performRequest(mux *http.ServeMux, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func authReqWithToken(req *http.Request, token string) *http.Request {
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func createAPIUserForStore(t *testing.T, store storage.ExtendedStorage, username string) (string, string) {
	t.Helper()

	passwordHash, err := auth.HashPassword("test-password")
	if err != nil {
		t.Fatalf("failed to hash test password: %v", err)
	}

	user := &model.User{
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: passwordHash,
		IsActive:     true,
		IsAdmin:      false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("failed to create API test user: %v", err)
	}

	token := username + "-api-token"
	apiKey := &model.APIKey{
		Name:      username + "-key",
		Key:       auth.HashToken(token),
		UserID:    user.ID,
		CreatedAt: time.Now(),
	}
	if err := store.CreateAPIKey(context.Background(), apiKey); err != nil {
		t.Fatalf("failed to create API test key: %v", err)
	}

	return user.ID, token
}

func (e *extendedTestHandler) createAPIUser(t *testing.T, username string) (string, string) {
	return createAPIUserForStore(t, e.store, username)
}
