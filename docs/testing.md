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
│   ├── scanner.go
│   └── scanner_test.go           # Scanner unit tests
└── ...
```

### Test Categories

1. **Unit Tests**: Test individual functions and methods in isolation
2. **Integration Tests**: Test component interactions and full workflows
3. **End-to-End Tests**: Test complete user scenarios (planned)

## Unit Tests

Unit tests focus on testing individual functions, methods, and components in isolation.

### Naming Convention

- Test files: `*_test.go`
- Test functions: `TestFunctionName` or `TestType_Method`
- Helper functions: `testHelperName` (lowercase)

### Example Unit Test

```go
func TestAuthMiddleware_ValidToken(t *testing.T) {
    called := false
    handler := AuthMiddleware("secret-token", func(w http.ResponseWriter, r *http.Request) {
        called = true
        w.WriteHeader(http.StatusOK)
    })

    r := httptest.NewRequest("GET", "/", nil)
    r.Header.Set("Authorization", "Bearer secret-token")
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
- `internal/discovery/scanner_test.go` - Network scanning logic
- `internal/config/config_test.go` - Configuration management

## Integration Tests

Integration tests verify that multiple components work together correctly.

### Build Tags

Integration tests use build tags to separate them from unit tests:

```go
//go:build !short

package api

import (
    "testing"
    // ...
)
```

### Database Integration Tests

Integration tests use in-memory SQLite databases for isolation:

```go
func setupTestStorage(t *testing.T) storage.ExtendedStorage {
    t.Helper()
    store, err := storage.NewSQLiteStorage(":memory:")
    if err != nil {
        t.Fatalf("failed to create test storage: %v", err)
    }
    t.Cleanup(func() { store.Close() })
    return store
}
```

### API Integration Tests

API integration tests create full HTTP servers with middleware:

```go
func setupIntegrationServer(t *testing.T, authToken string) *httptest.Server {
    t.Helper()
    store := setupTestStorage(t)
    
    mux := http.NewServeMux()
    RegisterHandlers(mux, store, authToken)
    
    return httptest.NewServer(mux)
}
```

### Key Integration Test Files

- `internal/api/integration_test.go` - Full API workflow tests
- `internal/storage/integration_test.go` - Cross-component storage tests

## Running Tests

### Basic Test Commands

```bash
# Run all tests
make test

# Run only unit tests (short tests)
make test-short

# Run tests with race detection
make test-race

# Show test coverage
make test-coverage
```

### Manual Test Commands

```bash
# Run all tests with verbose output
go test -v ./...

# Run only unit tests
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
# Skip integration tests
go test -v -short ./...

# Run only integration tests
go test -v -run Integration ./...

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
make test-coverage

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
                Name: "test-net",
                CIDR: "192.168.1.0/24",
            },
            wantErr: false,
        },
        {
            name: "invalid CIDR",
            network: &model.Network{
                Name: "test-net",
                CIDR: "invalid",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateNetwork(tt.network)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateNetwork() error = %v, wantErr %v", err, tt.wantErr)
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
        ID:       uuid.Must(uuid.NewV7()).String(),
        Name:     "test-device",
        IP:       "192.168.1.100",
        Type:     "server",
        Status:   "active",
    }
}
```

#### Cleanup

```go
func TestDatabaseOperations(t *testing.T) {
    store := setupTestStorage(t)
    t.Cleanup(func() {
        store.Close()
    })
    
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
func TestCreateDevice_InvalidInput(t *testing.T) {
    store := setupTestStorage(t)
    
    // Test with nil device
    err := store.CreateDevice(nil)
    if err == nil {
        t.Error("expected error for nil device")
    }
    
    // Test with invalid device
    device := &model.Device{Name: ""} // Invalid: empty name
    err = store.CreateDevice(device)
    if err == nil {
        t.Error("expected error for invalid device")
    }
}
```

## CI/CD Integration

### GitHub Actions

Tests run automatically on:
- Pull requests
- Pushes to main branch
- Release tags

### Test Pipeline

```yaml
- name: Run Tests
  run: |
    make test-short
    make test-race
    make test-coverage
```

### Quality Gates

- All tests must pass
- Coverage must not decrease
- No race conditions detected
- Security scans pass

## Test Data Management

### Test Fixtures

Store test data in `testdata/` directories:

```
internal/api/testdata/
├── devices.json
├── networks.json
└── discovery_results.json
```

### Database Migrations

Test database migrations separately:

```go
func TestMigrations(t *testing.T) {
    db := setupTestDB(t)
    
    // Test each migration step
    for i, migration := range migrations {
        t.Run(fmt.Sprintf("migration_%d", i), func(t *testing.T) {
            err := migration.Up(db)
            if err != nil {
                t.Fatalf("migration %d failed: %v", i, err)
            }
        })
    }
}
```

## Performance Testing

### Benchmark Tests

```go
func BenchmarkDeviceSearch(b *testing.B) {
    store := setupBenchmarkStorage(b)
    
    // Insert test data
    for i := 0; i < 1000; i++ {
        device := &model.Device{
            ID:   uuid.Must(uuid.NewV7()).String(),
            Name: fmt.Sprintf("device-%d", i),
        }
        store.CreateDevice(device)
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := store.SearchDevices("device-500")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Load Testing

Use external tools for load testing:

```bash
# Example with hey
hey -n 1000 -c 10 http://localhost:8080/api/v1/devices
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
4. **Slow Tests**: Use build tags to separate

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