# Enterprise Phase 6 Security Review

**Review Date:** 2026-01-22
**Reviewer:** Security Review (Automated)
**Phase:** Enterprise Phase 6 - Enterprise Server
**Status:** PASSED with recommendations

## Scope

This review covers Enterprise Phase 6 tasks:
- E6-001: Enterprise Server Entry Point (`cmd/rackd-enterprise/main.go`)
- E6-002: Enterprise API Handlers (`internal/features/advanced_scanning.go`)
- E6-003: Enterprise MCP Tools (same file)

Also reviewed supporting files:
- `pkg/server/server.go` (OSS public server package)
- `internal/server/server.go` (OSS internal server)

## Executive Summary

Enterprise Phase 6 implements the enterprise server entry point and integrates the Advanced Scanning feature from Phase 5. The implementation follows the established OSS/Enterprise separation pattern correctly. Security controls from Phase 5 (credential encryption, DTO pattern) are properly utilized.

**Overall Assessment:** PASSED

| Category | Status | Findings |
|----------|--------|----------|
| Authentication | ✅ PASS | Inherits OSS auth middleware |
| Authorization | ⚠️ LOW | No role-based access control |
| Encryption Key Management | ✅ PASS | Required in production, dev-mode for testing |
| Input Validation | ✅ PASS | Model validation enforced |
| Error Handling | ⚠️ LOW | Some errors expose internal details |
| Secrets Management | ✅ PASS | Credentials encrypted, DTOs used |

## Detailed Findings

### SEC-E6-001: Encryption Key Fallback (RESOLVED)

**Location:** `cmd/rackd-enterprise/main.go:113-125`

**Original Issue:** When `ENCRYPTION_KEY` is not set, a random key is generated. This means credentials encrypted in one session cannot be decrypted after restart.

**Resolution:** Added `--dev-mode` flag. Without this flag, the server now fails to start if `ENCRYPTION_KEY` is not set:

```go
func getEncryptionKey(devMode bool) ([]byte, error) {
    keyHex := os.Getenv("ENCRYPTION_KEY")
    if keyHex == "" {
        if !devMode {
            return nil, fmt.Errorf("ENCRYPTION_KEY environment variable is required (use --dev-mode to allow random key for development)")
        }
        // Only in dev mode: generate random key with warning
        fmt.Fprintln(os.Stderr, "Warning: ENCRYPTION_KEY not set - generating random key...")
        // ...
    }
    // ...
}
```

**Status:** ✅ RESOLVED - Production deployments now require `ENCRYPTION_KEY`. Development can use `--dev-mode` flag.

---

### SEC-E6-002: No Role-Based Access Control (LOW)

**Location:** `internal/features/advanced_scanning.go` (all handlers)

**Issue:** All credential and scan profile endpoints are accessible to any authenticated user. No RBAC checks are performed.

**Risk:** Any authenticated user can:
- View/modify/delete credentials
- Create/modify scan profiles
- Schedule scans on any network

**Recommendation:** Future enhancement - implement RBAC when enterprise auth features are added. For now, document that all authenticated users have full access.

**Mitigation:** This is expected for current phase. RBAC is an enterprise feature planned for later phases.

---

### SEC-E6-003: Error Message Information Disclosure (LOW)

**Location:** `internal/features/advanced_scanning.go` (multiple handlers)

**Issue:** Some error messages expose internal details:

```go
func (f *AdvancedScanningFeature) mcpAdvancedScan(...) (*mcp.ToolResponse, error) {
    // ...
    if err != nil {
        return nil, mcp.NewToolErrorInternal("network not found: " + err.Error())
    }
    // ...
    if err != nil {
        return nil, mcp.NewToolErrorInternal("scan failed: " + err.Error())
    }
}
```

**Risk:** Internal error details could reveal system information to attackers.

**Recommendation:** Log full errors server-side, return generic messages to clients:
```go
if err != nil {
    log.Error("network lookup failed", "error", err)
    return nil, mcp.NewToolErrorInternal("network not found")
}
```

---

### SEC-E6-004: Credential Response DTO Correctly Used (PASS)

**Location:** `internal/features/advanced_scanning.go:152-157, 163-164`

**Verification:** All credential list/get operations return `CredentialResponse` DTO, not raw `Credential`:

```go
func (f *AdvancedScanningFeature) listCredentials(w http.ResponseWriter, r *http.Request) {
    creds, err := f.credStore.List(datacenterID)
    // ...
    responses := make([]model.CredentialResponse, len(creds))
    for i, c := range creds {
        responses[i] = c.ToResponse()
    }
    writeJSON(w, responses)
}
```

**Status:** ✅ Correctly implemented - sensitive fields never exposed via API.

---

### SEC-E6-005: Feature Registration Security (PASS)

**Location:** `pkg/server/server.go`, `internal/server/server.go`

**Verification:** Feature registration follows secure patterns:
1. Features registered via typed interface, not arbitrary code execution
2. Route registration uses standard `http.ServeMux`
3. MCP tools registered through typed interface
4. UI configuration limited to predefined operations

```go
type Feature interface {
    Name() string
    RegisterRoutes(mux *http.ServeMux)
    RegisterMCPTools(mcpServer *mcp.Server)
    ConfigureUI(builder UIConfigBuilder)
}
```

**Status:** ✅ Clean separation prevents enterprise code from bypassing OSS security controls.

---

### SEC-E6-006: Storage Adapter Type Safety (PASS)

**Location:** `pkg/server/server.go:70-300`

**Verification:** `StorageAdapter` properly converts between internal and public types without exposing internal implementation details:

```go
func (s *StorageAdapter) GetDevice(id string) (*rackd.Device, error) {
    d, err := s.internal.GetDevice(id)
    if err != nil {
        return nil, err
    }
    return convertDeviceToPublic(d), nil
}
```

**Status:** ✅ Type conversion prevents leaking internal types to enterprise code.

---

### SEC-E6-007: Scheduled Scan Worker Security (LOW)

**Location:** `internal/worker/scheduled.go`

**Issue:** Scheduled scans run with the service's credentials, not the user who created them.

**Risk:** A user could schedule a scan and then have their access revoked, but the scan continues running.

**Recommendation:** Future enhancement - store creator ID with scheduled scans and validate permissions at execution time.

**Mitigation:** Acceptable for current phase. Document that scheduled scans run with service-level permissions.

---

### SEC-E6-008: SSH Host Key Verification (PASS)

**Location:** `internal/discovery/ssh.go:27-50, 143-157`

**Verification:** Trust-on-first-use (TOFU) host key verification is implemented:

```go
func (s *SSHScanner) trustOnFirstUseCallback(host string) ssh.HostKeyCallback {
    return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
        knownKey, err := s.hostKeyStore.Get(host)
        if err != nil {
            return fmt.Errorf("host key lookup failed: %w", err)
        }
        if knownKey == nil {
            return s.hostKeyStore.Store(host, key)
        }
        if string(knownKey.Marshal()) != string(key.Marshal()) {
            return fmt.Errorf("host key mismatch for %s: possible MITM attack", host)
        }
        return nil
    }
}
```

**Status:** ✅ TOFU provides reasonable security for automated scanning. Host key changes are detected and rejected.

**Note:** In-memory store means keys are lost on restart. Consider persistent storage for production.

---

### SEC-E6-009: SNMP Security Warning (INFORMATIONAL)

**Location:** `internal/discovery/snmp.go:52-55`

**Observation:** Code includes appropriate security warning for SNMPv2c:

```go
case "snmp_v2c":
    // WARNING: SNMPv2c transmits community string in cleartext.
    // Use only on trusted networks. Consider SNMPv3 for production.
    client.Version = gosnmp.Version2c
```

**Status:** ✅ Good practice - security implications documented in code.

---

## Architecture Security Assessment

### OSS/Enterprise Separation

The separation is correctly maintained:

1. **OSS has no enterprise knowledge:** Internal server code uses generic `Feature` interface
2. **Enterprise extends via public API:** Uses `pkg/rackd` and `pkg/server` packages
3. **No circular dependencies:** Enterprise imports OSS, never reverse
4. **Type safety maintained:** `StorageAdapter` converts types at boundary

### Authentication Flow

```
Request → OSS Middleware (auth check) → Feature Handler → Storage
```

Enterprise features inherit OSS authentication automatically through the shared `http.ServeMux`.

### Credential Security Flow

```
API Request → Handler → CredentialResponse DTO (no secrets)
                ↓
           credStore.Create/Update
                ↓
           Encryptor.Encrypt (AES-256-GCM)
                ↓
           SQLite (encrypted values)
```

Secrets are encrypted at rest and never exposed via API.

## Recommendations Summary

| ID | Severity | Recommendation | Status |
|----|----------|----------------|--------|
| SEC-E6-001 | ~~MEDIUM~~ | ~~Fail startup without ENCRYPTION_KEY in production~~ | ✅ RESOLVED |
| SEC-E6-002 | LOW | Document current access model, plan RBAC | Open |
| SEC-E6-003 | LOW | Sanitize error messages in responses | Open |
| SEC-E6-007 | LOW | Document scheduled scan permission model | Open |

## Compliance Checklist

- [x] Sensitive data encrypted at rest (AES-256-GCM)
- [x] Sensitive data not exposed in API responses (DTO pattern)
- [x] Authentication inherited from OSS
- [x] Input validation on all models
- [x] SSH host key verification implemented
- [x] SNMPv3 support for secure SNMP
- [ ] RBAC (planned for future phase)
- [ ] Audit logging (planned for future phase)

## Conclusion

Enterprise Phase 6 successfully integrates the Advanced Scanning feature with the enterprise server. Security controls established in Phase 5 are properly utilized. The OSS/Enterprise separation is maintained correctly, preventing enterprise code from bypassing OSS security controls.

The main area for improvement is encryption key management - the random key fallback should be restricted to development environments only.

**Result: PASSED**
