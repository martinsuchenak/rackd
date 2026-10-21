# Testing

This document covers the testing strategy, structure, and guidelines for the Rackd project.

## Test Structure

Rackd follows Go's standard testing conventions with additional patterns for comprehensive coverage:

```
internal/
├── api/
│   ├── handlers.go
│   ├── handlers_test.go           # Unit tests for handlers
│   ├── middleware_test.go         # Middleware unit tests
│   └── integration_test.go        # API integration tests
├── storage/
│   ├── sqlite.go
│   ├── sqlite_test.go            # Storage unit tests
│   ├── migrations_test.go        # Migration tests
│   └── integration_test.go       # Storage integration tests
├── discovery/
│   ├── unified_scanner.go
│   └── scanner_test.go           # Scanner unit tests
└── ...
```

### Test Categories

1. **Unit Tests**: Test individual functions and methods in isolation
2. **Integration Tests**: Test component interactions and full workflows
3. **End-to-End Tests**: Playwright tests of the web UI in `webui/e2e`, run with `task test:e2e`

## Unit Tests

Unit tests focus on testing individual functions, methods, and components in isolation.

### Naming Convention

- Test files: `*_test.go`
- Test functions: `TestFunctionName` or `TestType_Method`
- Helper functions: `testHelperName` (lowercase)

### Example Unit Test

API keys belong to users, so the test creates a user and a key for it first:

```go
func TestAuthMiddleware_ValidToken(t *testing.T) {
    store := storagetest.New(t)
    defer func() { _ = store.Close() }()

    passwordHash, _ := auth.HashPassword("test-password")
    user := &model.User{
        ID: "test-user-id", Username: "testuser", Email: "test@example.com",
        PasswordHash: passwordHash, IsActive: true, IsAdmin: true,
        CreatedAt: time.Now(), UpdatedAt: time.Now(),
    }
    if err := store.CreateUser(context.Background(), user); err != nil {
        t.Fatalf("Failed to create test user: %v", err)
    }

    keySecret := "secret-token"
    key := &model.APIKey{Name: "test-key", Key: auth.HashToken(keySecret), UserID: user.ID}
    if err := store.CreateAPIKey(context.Background(), key); err != nil {
        t.Fatalf("Failed to create API key: %v", err)
    }

    called := false
    handler := AuthMiddleware(store, func(w http.ResponseWriter, r *http.Request) {
        called = true
        w.WriteHeader(http.StatusOK)
    })

    r := httptest.NewRequest("GET", "/", nil)
    r.Header.Set("Authorization", "Bearer "+keySecret)
    w := httptest.NewRecorder()

    handler(w, r)

    if !called {
        t.Error("handler was not called")
    }
    if w.Code != http.StatusOK {
        t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
    }
}
```

### Key Unit Test Files

- `internal/api/middleware_test.go` - Authentication and security middleware
- `internal/api/handlers_test.go` - HTTP handler functions
- `internal/storage/sqlite_test.go` - Database operations
- `internal/discovery/scanner_test.go` - Network scanning logic (`unified_scanner.go`)
- `internal/config/config_test.go` - Configuration management

## Integration Tests

Integration tests verify that multiple components work together correctly.

### Build Tags

The integration, security and benchmark test files in `internal/api` and `internal/storage` start with a build constraint:

```go
//go:build !short

package api
```

`go test -short` doesn't set that tag, so these files always build and run. Building with `-tags short` doesn't work either, because other tests in `internal/api` use the helpers in `integration_test.go`. To skip slow tests under `-short`, call `testing.Short()`, as the storage integration tests do.

### Database Integration Tests

Tests get their database from `internal/storage/storagetest`. `storagetest.New(t)` migrates a template database once per test binary and gives each test a private copy of it:

```go
store := storagetest.New(t)
t.Cleanup(func() { _ = store.Close() })
```

For a quick throwaway database, `storage.NewSQLiteStorage(":memory:")` also works.

### API Integration Tests

API integration tests wire a handler with sessions and services, and register every route on a mux (`newTestServer` in `internal/api/integration_test.go`):

```go
store := storagetest.New(t)
t.Cleanup(func() { _ = store.Close() })

sm := auth.NewSessionManager(24*time.Hour, nil)
t.Cleanup(func() { sm.Stop() })

svc := service.NewServices(store, sm, nil)

h := NewHandler(store, nil,
    WithSessionManager(sm),
    WithCookieConfig(false, 24*time.Hour),
    WithServices(svc),
)

mux := http.NewServeMux()
h.RegisterRoutes(mux)
```

### Key Integration Test Files

- `internal/api/integration_test.go` - Full API workflow tests
- `internal/storage/integration_test.go` - Cross-component storage tests

## Running Tests

### Basic Test Commands

```bash
# Run all tests
task test

# Run short tests (skips tests that check testing.Short())
task test:short

# Run tests with race detection
task test:race

# Show test coverage
task test:cover
```

### Manual Test Commands

```bash
# Run all tests with verbose output
go test -v ./...

# Run short tests
go test -v -short ./...

# Run specific package tests
go test -v ./internal/api

# Run specific test function
go test -v -run TestAuthMiddleware ./internal/api

# Run with race detection
go test -v -race ./...

# Generate coverage report
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Test Filtering

```bash
# Skip the storage integration tests (they check testing.Short())
go test -v -short ./...

# Run the API full-stack integration tests
go test -v -run FullStack ./internal/api

# Run tests matching pattern
go test -v -run "Test.*Handler" ./internal/api
```

## Test Coverage

### Coverage Goals

- **Unit Tests**: >80% line coverage for core business logic
- **Integration Tests**: Cover all major user workflows
- **Critical Paths**: 100% coverage for security and data integrity

### Viewing Coverage

```bash
# Generate coverage report
task test:cover

# View coverage in browser
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Coverage by Package

Current coverage targets:
- `internal/api`: >85%
- `internal/storage`: >90%
- `internal/discovery`: >80%
- `internal/config`: >85%

## Writing Tests

### Test Setup Patterns

#### Table-Driven Tests

`ValidateNetwork` (in `internal/service`) returns a list of `ValidationErrors`, empty when the network is valid:

```go
func TestValidateNetwork(t *testing.T) {
    tests := []struct {
        name    string
        network *model.Network
        wantErr bool
    }{
        {
            name: "valid network",
            network: &model.Network{
                Name:   "test-net",
                Subnet: "192.168.1.0/24",
            },
            wantErr: false,
        },
        {
            name: "invalid CIDR",
            network: &model.Network{
                Name:   "test-net",
                Subnet: "invalid",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            errs := ValidateNetwork(tt.network)
            if (len(errs) > 0) != tt.wantErr {
                t.Errorf("ValidateNetwork() errors = %v, wantErr %v", errs, tt.wantErr)
            }
        })
    }
}
```

#### Test Helpers

```go
func setupTestDevice(t *testing.T) *model.Device {
    t.Helper()
    return &model.Device{
        ID:     uuid.Must(uuid.NewV7()).String(),
        Name:   "test-device",
        Status: model.DeviceStatusActive,
        Addresses: []model.Address{
            {IP: "192.168.1.100", Type: "ipv4"},
        },
    }
}
```

#### Cleanup

```go
func TestDatabaseOperations(t *testing.T) {
    store := storagetest.New(t)
    t.Cleanup(func() { _ = store.Close() })

    // Test implementation
}
```

### Mock Objects

For external dependencies, use interfaces and mocks:

```go
type mockScanner struct {
    scanFunc func(context.Context, *model.Network, string) (*model.DiscoveryScan, error)
}

func (m *mockScanner) Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
    if m.scanFunc != nil {
        return m.scanFunc(ctx, network, scanType)
    }
    return nil, errors.New("not implemented")
}
```

### Error Testing

Always test error conditions:

```go
func TestCreateDevice_NilDevice(t *testing.T) {
    store := storagetest.New(t)
    t.Cleanup(func() { _ = store.Close() })

    if err := store.CreateDevice(context.Background(), nil); err == nil {
        t.Error("expected error for nil device")
    }
}
```

Field validation, such as a required name, happens in the service layer (`internal/service`), so test invalid input there rather than against storage.

## CI/CD Integration

### GitHub Actions

The `ci` workflow (`.github/workflows/ci.yml`) runs on:
- Pull requests
- Pushes to main branch

The release workflow, triggered by a tag, builds and publishes with GoReleaser and doesn't rerun the tests; `task release` waits for CI to pass on the commit before tagging it.

### Test Pipeline

CI runs these jobs (the Go jobs build the UI with `task ui:build` first):

- **Go**: `gofmt` check, `go mod tidy` check, `task vet`, `task lint`, `go test ./... -count=1`, `govulncheck ./...`
- **Race**: `go test ./... -count=1 -race -timeout 30m`
- **Cross-compile**: `go build` for each release platform
- **Web UI**: `bun run typecheck`, `bun run validate:types` (types against the OpenAPI spec), `bun test`
- **Docker**: builds the image

CI sets `RACKD_BCRYPT_COST=6` to keep the suite fast.

### Quality Gates

- All tests must pass
- No race conditions detected
- No known vulnerabilities (`govulncheck`)

## Test Data Management

### Test Fixtures

Go tests keep their fixtures in a `testdata/` directory next to the package, such as `internal/storage/testdata/fixtures/` (a legacy v0.0.21 database for migration tests).

The top-level `testdata/` holds sample data for a running server (`devices.json`, `networks.json`, `datacenters.json` and more), loaded with `testdata/load-testdata.sh`; see `testdata/README.md`.

### Database Migrations

`internal/storage/migrations_test.go` covers migrations: it runs them all on a fresh database (`RunMigrations`), checks they are idempotent, reverts every Down step, upgrades the legacy fixture, and checks that a backup is taken before rollback and upgrade:

```go
func TestRunMigrations(t *testing.T) {
    db := openMemDB(t)
    if err := RunMigrations(context.Background(), db); err != nil {
        t.Fatalf("RunMigrations failed: %v", err)
    }
    if got := countCoreRecords(t, db); got != len(migrations) {
        t.Errorf("expected %d migration records, got %d", len(migrations), got)
    }
}
```

## Performance Testing

### Benchmark Tests

```go
func BenchmarkDeviceSearch(b *testing.B) {
    store := storagetest.New(b)
    b.Cleanup(func() { _ = store.Close() })
    ctx := context.Background()

    // Insert test data
    for i := 0; i < 1000; i++ {
        device := &model.Device{
            ID:   uuid.Must(uuid.NewV7()).String(),
            Name: fmt.Sprintf("device-%d", i),
        }
        _ = store.CreateDevice(ctx, device)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := store.SearchDevices(ctx, "device-500")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Load Testing

`tools/loadtest` seeds a disposable server with test data and measures API latency and throughput; see [Performance](performance.md). External tools work too, with an API key:

```bash
# Example with hey
hey -n 1000 -c 10 -H "Authorization: Bearer $RACKD_TOKEN" http://localhost:8080/api/v1/devices
```

## Debugging Tests

### Verbose Output

```bash
go test -v -run TestSpecificTest ./internal/api
```

### Test Debugging

```go
func TestDebugExample(t *testing.T) {
    if testing.Verbose() {
        t.Logf("Debug info: %+v", someStruct)
    }
    
    // Use t.Helper() in helper functions
    // Use t.Cleanup() for cleanup
    // Use t.Skip() to skip tests conditionally
}
```

### Common Issues

1. **Race Conditions**: Use `go test -race`
2. **Resource Leaks**: Ensure proper cleanup
3. **Flaky Tests**: Use deterministic test data
4. **Slow Tests**: Skip them under `testing.Short()`

## Best Practices

### Do's

- Write tests before or alongside code
- Use descriptive test names
- Test both success and failure cases
- Use table-driven tests for multiple scenarios
- Keep tests simple and focused
- Use helper functions to reduce duplication
- Clean up resources properly

### Don'ts

- Don't test implementation details
- Don't use real external services in tests
- Don't ignore test failures
- Don't write overly complex tests
- Don't test third-party code
- Don't use global state in tests

### Test Organization

```go
func TestPackage(t *testing.T) {
    t.Run("group1", func(t *testing.T) {
        t.Run("case1", func(t *testing.T) { /* test */ })
        t.Run("case2", func(t *testing.T) { /* test */ })
    })
    
    t.Run("group2", func(t *testing.T) {
        t.Run("case1", func(t *testing.T) { /* test */ })
        t.Run("case2", func(t *testing.T) { /* test */ })
    })
}
```

## Continuous Improvement

### Metrics to Track

- Test coverage percentage
- Test execution time
- Number of flaky tests
- Test maintenance overhead

### Regular Reviews

- Review test coverage reports monthly
- Update test data and fixtures
- Refactor slow or complex tests
- Add tests for new bug fixes

### Tools Integration

- Coverage reporting: `go tool cover`
- Race detection: `go test -race`
- Benchmarking: `go test -bench`
- Profiling: `go test -cpuprofile`