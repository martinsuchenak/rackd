# Testing Strategy

This document covers the testing approach and patterns for Rackd.

## Test Structure

```
internal/
├── api/
│   ├── device_handlers_test.go
│   ├── network_handlers_test.go
│   └── ...
├── storage/
│   ├── sqlite_test.go
│   └── ...
└── discovery/
    └── scanner_test.go
```

## Running Tests

```bash
# All tests with coverage
make test

# Short tests (skip integration)
make test-short

# Coverage report
make test-coverage
```

## Test Patterns

### Handler Tests

```go
func TestCreateDevice(t *testing.T) {
    // Setup
    store := setupTestStorage(t)
    handler := api.NewHandler(store)

    // Test
    device := &model.Device{Name: "test-server"}
    req := httptest.NewRequest("POST", "/api/devices", toJSON(device))
    rec := httptest.NewRecorder()

    handler.createDevice(rec, req)

    // Assert
    assert.Equal(t, http.StatusCreated, rec.Code)

    var created model.Device
    json.Unmarshal(rec.Body.Bytes(), &created)
    assert.NotEmpty(t, created.ID)
    assert.Equal(t, "test-server", created.Name)
}
```

### Storage Tests

```go
func TestDeviceStorage(t *testing.T) {
    store := setupTestStorage(t)

    t.Run("Create", func(t *testing.T) {
        device := &model.Device{
            Name: "test-device",
            Tags: []string{"web", "prod"},
        }
        err := store.CreateDevice(device)
        assert.NoError(t, err)
        assert.NotEmpty(t, device.ID)
    })

    t.Run("Get", func(t *testing.T) {
        device, err := store.GetDevice(createdID)
        assert.NoError(t, err)
        assert.Equal(t, "test-device", device.Name)
    })

    t.Run("NotFound", func(t *testing.T) {
        _, err := store.GetDevice("nonexistent")
        assert.ErrorIs(t, err, storage.ErrDeviceNotFound)
    })
}
```

### Discovery Tests

```go
func TestScanner(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping network test in short mode")
    }

    store := setupTestStorage(t)
    cfg := &config.Config{
        DiscoveryTimeout:       time.Second,
        DiscoveryMaxConcurrent: 5,
    }
    scanner := discovery.NewScanner(store, cfg)

    network := &model.Network{
        ID:     "test-network",
        Subnet: "192.168.1.0/28", // Small subnet for testing
    }

    scan, err := scanner.Scan(context.Background(), network, model.ScanTypeQuick)
    assert.NoError(t, err)
    assert.Equal(t, model.ScanStatusPending, scan.Status)
}
```

## Test Helpers

```go
// setupTestStorage creates an in-memory SQLite database for testing
func setupTestStorage(t *testing.T) storage.ExtendedStorage {
    t.Helper()

    store, err := storage.NewSQLiteStorage(":memory:")
    if err != nil {
        t.Fatalf("failed to create test storage: %v", err)
    }

    t.Cleanup(func() {
        store.Close()
    })

    return store
}

// toJSON converts a struct to io.Reader for request body
func toJSON(v interface{}) io.Reader {
    data, _ := json.Marshal(v)
    return bytes.NewReader(data)
}
```

## Test Categories

### Unit Tests

- Model validation
- Storage operations
- Business logic functions
- Helper utilities

### Integration Tests

- Full API request/response cycles
- Database migrations
- Discovery scanning (network required)

### End-to-End Tests

- CLI commands
- MCP tool calls
- Web UI interactions (if applicable)

## Coverage Goals

| Package | Target Coverage |
|---------|-----------------|
| `internal/api` | 80%+ |
| `internal/storage` | 90%+ |
| `internal/discovery` | 70%+ |
| `internal/model` | 60%+ |

## Mocking

Use interfaces for easy mocking:

```go
type MockStorage struct {
    devices map[string]*model.Device
}

func (m *MockStorage) GetDevice(id string) (*model.Device, error) {
    if device, ok := m.devices[id]; ok {
        return device, nil
    }
    return nil, storage.ErrDeviceNotFound
}
```

## CI Integration

Tests run automatically on:
- Pull requests
- Main branch pushes
- Release tags

GitHub Actions workflow:

```yaml
test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.25'
    - run: make test
    - uses: codecov/codecov-action@v4
      with:
        file: coverage.out
```
