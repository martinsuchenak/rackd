# Phase 6 Security Review: Server Assembly

**Date:** 2026-01-22  
**Reviewer:** AI Security Review  
**Status:** PASSED  
**Files Reviewed:**
- `internal/server/server.go`
- `internal/server/server_test.go`
- `internal/ui/ui.go`
- `internal/ui/ui_test.go`
- `internal/ui/assets/*` (placeholder files)

---

## Summary

Phase 6 implements the server entry point that wires together all components (API, MCP, discovery scheduler, UI) and the embedded UI handler with SPA fallback. The implementation follows security best practices.

**Overall Assessment:** PASSED

---

## Findings

### HIGH Priority

None identified.

### MEDIUM Priority

None identified.

### LOW Priority

#### SEC-P6-001: Health Endpoint Unauthenticated

**Location:** `server.go:72-75`

**Issue:** The `/healthz` endpoint is not protected by authentication.

**Risk:** LOW - This is intentional and standard practice. Health endpoints must be accessible for load balancers and orchestration systems (Kubernetes, Nomad) to check service health.

**Status:** ACCEPTABLE - No action required.

---

#### SEC-P6-002: UI Config Endpoint Unauthenticated

**Location:** `server.go:68`

**Issue:** The `/api/config` endpoint is registered outside the authenticated handler routes.

**Risk:** LOW - The UI config endpoint returns non-sensitive information (edition, feature flags, nav items). It must be accessible before authentication to allow the UI to render properly.

**Status:** ACCEPTABLE - No action required. The endpoint does not expose sensitive data.

---

## Security Controls Verified

### Authentication Warnings

✓ **VERIFIED:** Server logs warnings when auth tokens are not configured:
```go
if cfg.APIAuthToken == "" {
    log.Warn("API_AUTH_TOKEN not set - API is unauthenticated")
}
if cfg.MCPAuthToken == "" {
    log.Warn("MCP_AUTH_TOKEN not set - MCP endpoint is unauthenticated")
}
```

### Security Headers

✓ **VERIFIED:** All responses pass through `api.SecurityHeaders` middleware:
```go
server := &http.Server{
    Handler: api.SecurityHeaders(mux),
    ...
}
```

This applies:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy: default-src 'self'...`
- `Strict-Transport-Security` (for TLS connections)

### HTTP Server Timeouts

✓ **VERIFIED:** Proper timeouts configured to prevent slowloris attacks:
```go
server := &http.Server{
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

### Graceful Shutdown

✓ **VERIFIED:** Server handles SIGINT/SIGTERM with 30-second timeout:
- Stops discovery scheduler first
- Gracefully drains HTTP connections
- Prevents abrupt connection termination

### Embedded UI Security

✓ **VERIFIED:** `embed.FS` is inherently safe against path traversal attacks:
- Only serves files that were embedded at compile time
- Cannot access files outside the embedded filesystem
- `fs.ReadFile` returns error for any path not in the embed

### Feature Interface Security

✓ **VERIFIED:** The Feature interface is minimal and does not expose internal types:
```go
type Feature interface {
    Name() string
    RegisterRoutes(mux *http.ServeMux)
    RegisterMCPTools(mcpServer *mcp.Server)
    ConfigureUI(builder *api.UIConfigBuilder)
}
```

Enterprise features register their own routes and are responsible for their own authentication (can use the same auth middleware).

---

## Architecture Decision: P6-001 Skipped

**Decision:** Skip P6-001 (Define Enterprise Interfaces in OSS)

**Security Rationale:** Defining enterprise-specific interfaces (AuthProvider, RBACChecker, AuditLogger, etc.) in the OSS repository would:

1. **Leak enterprise concepts** - OSS users would see interfaces for features they cannot use
2. **Create coupling** - OSS would need to evolve interfaces based on enterprise needs
3. **Violate separation** - The architectural principle states OSS should have no knowledge of enterprise features

The Feature interface provides a clean, minimal extension point without exposing enterprise internals.

---

## Recommendations

### For Production Deployment

1. **Always set auth tokens** - Never run in production without `API_AUTH_TOKEN` and `MCP_AUTH_TOKEN`
2. **Use TLS** - Deploy behind a TLS-terminating proxy or configure TLS directly to enable HSTS
3. **Restrict listen address** - Use `127.0.0.1:8080` if only local access is needed

---

## Test Coverage

| File | Coverage | Notes |
|------|----------|-------|
| `server.go` | Feature interface tested | Full server test requires integration test |
| `ui.go` | 100% | All routes and SPA fallback tested |

---

## Conclusion

Phase 6 implementation follows security best practices:
- Authentication warnings for operators
- Security headers on all responses
- Proper HTTP timeouts
- Graceful shutdown
- Safe embedded filesystem
- Clean separation between OSS and enterprise

**Result:** PASSED
