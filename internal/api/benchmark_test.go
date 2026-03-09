//go:build !short

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

// benchEnv holds shared state for benchmarks.
type benchEnv struct {
	store  storage.ExtendedStorage
	mux    *http.ServeMux
	sm     *auth.SessionManager
	svc    *service.Services
	apiKey string
	userID string
}

func newBenchEnv(b *testing.B) *benchEnv {
	b.Helper()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		b.Fatalf("storage: %v", err)
	}

	sm := auth.NewSessionManager(24*time.Hour, nil)
	svc := service.NewServices(store, sm, nil)

	h := NewHandler(store, nil,
		WithSessionManager(sm),
		WithCookieConfig(false, 24*time.Hour),
		WithServices(svc),
	)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()

	// Create admin user with role
	if err := store.CreateInitialAdmin(ctx, "benchadmin", "bench@localhost", "Bench Admin", "benchpass123"); err != nil {
		b.Fatalf("create admin: %v", err)
	}
	user, err := store.GetUserByUsername(ctx, "benchadmin")
	if err != nil {
		b.Fatalf("get admin: %v", err)
	}

	// Create API key for auth
	rawToken := "bench-api-key-token"
	key := &model.APIKey{
		Name:   "bench-key",
		Key:    auth.HashToken(rawToken),
		UserID: user.ID,
	}
	if err := store.CreateAPIKey(ctx, key); err != nil {
		b.Fatalf("create api key: %v", err)
	}

	return &benchEnv{
		store:  store,
		mux:    mux,
		sm:     sm,
		svc:    svc,
		apiKey: rawToken,
		userID: user.ID,
	}
}

func (e *benchEnv) authReq(req *http.Request) *http.Request {
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	return req
}

// seedDevices creates n devices and returns their IDs.
func (e *benchEnv) seedDevices(b *testing.B, n int) []string {
	b.Helper()
	ctx := context.Background()
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		d := &model.Device{
			Name:        fmt.Sprintf("bench-device-%d", i),
			Hostname:    fmt.Sprintf("host-%d.bench.local", i),
			Description: fmt.Sprintf("Benchmark device number %d for performance testing", i),
			MakeModel:   "Dell R640",
			OS:          "Ubuntu 22.04",
			Status:      model.DeviceStatusActive,
			Tags:        []string{"bench", fmt.Sprintf("group-%d", i%10)},
			Addresses: []model.Address{
				{IP: fmt.Sprintf("10.0.%d.%d", i/256, i%256), Type: "management"},
			},
		}
		if err := e.store.CreateDevice(ctx, d); err != nil {
			b.Fatalf("seed device %d: %v", i, err)
		}
		ids[i] = d.ID
	}
	return ids
}

// --- Storage-layer benchmarks ---

func BenchmarkStorageCreateDevice(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := &model.Device{
			Name:     fmt.Sprintf("dev-%d", i),
			Hostname: fmt.Sprintf("h-%d.local", i),
			Status:   model.DeviceStatusActive,
			Tags:     []string{"bench"},
			Addresses: []model.Address{
				{IP: fmt.Sprintf("10.%d.%d.%d", i/65536%256, i/256%256, i%256), Type: "mgmt"},
			},
		}
		if err := env.store.CreateDevice(ctx, d); err != nil {
			b.Fatalf("create: %v", err)
		}
	}
}

func BenchmarkStorageGetDevice(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()
	ids := env.seedDevices(b, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := env.store.GetDevice(context.Background(), ids[i%len(ids)]); err != nil {
			b.Fatalf("get: %v", err)
		}
	}
}

func BenchmarkStorageListDevices(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()
	env.seedDevices(b, 500)

	for _, limit := range []int{10, 50, 100} {
		b.Run(fmt.Sprintf("limit=%d", limit), func(b *testing.B) {
			filter := &model.DeviceFilter{
				Pagination: model.Pagination{Limit: limit},
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := env.store.ListDevices(context.Background(), filter); err != nil {
					b.Fatalf("list: %v", err)
				}
			}
		})
	}
}

func BenchmarkStorageSearchDevices(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()
	env.seedDevices(b, 500)

	queries := []string{"bench", "host-42", "Dell", "Ubuntu"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		if _, err := env.store.SearchDevices(context.Background(), q); err != nil {
			b.Fatalf("search %q: %v", q, err)
		}
	}
}

func BenchmarkStorageListDevicesWithTagFilter(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()
	env.seedDevices(b, 500)

	filter := &model.DeviceFilter{
		Tags:       []string{"group-3"},
		Pagination: model.Pagination{Limit: 50},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := env.store.ListDevices(context.Background(), filter); err != nil {
			b.Fatalf("list with tags: %v", err)
		}
	}
}

// --- Auth benchmarks ---

func BenchmarkAPIKeyAuth(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := AuthenticateAPIKey(context.Background(), env.store, env.apiKey, "127.0.0.1", "bench")
		if err != nil {
			b.Fatalf("auth: %v", err)
		}
	}
}

func BenchmarkSessionValidation(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()

	sess, err := env.sm.CreateSession(env.userID, "benchadmin", true)
	if err != nil {
		b.Fatalf("create session: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := env.sm.GetSession(sess.Token); err != nil {
			b.Fatalf("get session: %v", err)
		}
	}
}

func BenchmarkPasswordHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := auth.HashPassword("benchmarkpassword123"); err != nil {
			b.Fatalf("hash: %v", err)
		}
	}
}

func BenchmarkPasswordVerify(b *testing.B) {
	hash, _ := auth.HashPassword("benchmarkpassword123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.VerifyPassword(hash, "benchmarkpassword123")
	}
}

func BenchmarkRBACPermissionCheck(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()

	ctx := service.WithCaller(context.Background(), &service.Caller{
		Type:   service.CallerTypeUser,
		UserID: env.userID,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		has, err := env.store.HasPermission(ctx, env.userID, "devices", "list")
		if err != nil {
			b.Fatalf("rbac: %v", err)
		}
		if !has {
			b.Fatal("expected permission")
		}
	}
}

// --- HTTP handler benchmarks ---

func BenchmarkHTTPListDevices(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()
	env.seedDevices(b, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := env.authReq(httptest.NewRequest("GET", "/api/devices?limit=50", nil))
		w := httptest.NewRecorder()
		env.mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("list: status %d", w.Code)
		}
	}
}

func BenchmarkHTTPGetDevice(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()
	ids := env.seedDevices(b, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := env.authReq(httptest.NewRequest("GET", "/api/devices/"+ids[i%len(ids)], nil))
		w := httptest.NewRecorder()
		env.mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("get: status %d, body: %s", w.Code, w.Body.String())
		}
	}
}

func BenchmarkHTTPCreateDevice(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body, _ := json.Marshal(map[string]any{
			"name":     fmt.Sprintf("http-dev-%d", i),
			"hostname": fmt.Sprintf("h-%d.local", i),
			"status":   "active",
		})
		req := env.authReq(httptest.NewRequest("POST", "/api/devices", bytes.NewReader(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			b.Fatalf("create: status %d, body: %s", w.Code, w.Body.String())
		}
	}
}

func BenchmarkHTTPSearchGlobal(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()
	env.seedDevices(b, 500)

	queries := []string{"bench", "host-42", "Dell", "Ubuntu"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		req := env.authReq(httptest.NewRequest("GET", "/api/search?q="+q, nil))
		w := httptest.NewRecorder()
		env.mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("search: status %d", w.Code)
		}
	}
}

// --- JSON serialization benchmarks ---

func BenchmarkJSONSerializeDeviceList(b *testing.B) {
	devices := make([]model.Device, 50)
	for i := range devices {
		devices[i] = model.Device{
			ID:          fmt.Sprintf("id-%d", i),
			Name:        fmt.Sprintf("device-%d", i),
			Hostname:    fmt.Sprintf("host-%d.local", i),
			Description: "A benchmark device for testing JSON serialization performance",
			MakeModel:   "Dell R640",
			OS:          "Ubuntu 22.04",
			Status:      model.DeviceStatusActive,
			Tags:        []string{"bench", "test", fmt.Sprintf("group-%d", i%5)},
			Addresses: []model.Address{
				{IP: fmt.Sprintf("10.0.0.%d", i), Type: "management"},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(devices); err != nil {
			b.Fatalf("marshal: %v", err)
		}
	}
}

// --- Middleware overhead benchmark ---

func BenchmarkMiddlewareChain(b *testing.B) {
	env := newBenchEnv(b)
	defer env.store.Close()
	env.seedDevices(b, 10)

	// Wrap mux with the same middleware chain as production
	var handler http.Handler = env.mux
	handler = LoggingMiddleware(SecurityHeaders(handler))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := env.authReq(httptest.NewRequest("GET", "/api/devices?limit=10", nil))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("middleware chain: status %d", w.Code)
		}
	}
}
