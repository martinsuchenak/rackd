# Phase 8 Security Review: CLI Client Library & Server Command (P8-001, P8-002)

**Date:** 2026-01-22  
**Reviewer:** AI Security Review  
**Status:** PASSED with recommendations  
**Files Reviewed:**
- `cmd/client/config.go`
- `cmd/client/http.go`
- `cmd/client/errors.go`
- `cmd/client/table.go`
- `cmd/server/server.go`

---

## Summary

Tasks P8-001 and P8-002 implement the CLI client library (config loading, HTTP client, error handling, output formatters) and the server command. The implementation is minimal and follows security best practices with a few low-priority recommendations.

**Overall Assessment:** PASSED

---

## Findings

### HIGH Priority

None identified.

### MEDIUM Priority

None identified.

### LOW Priority

#### SEC-P8-001: Config File Permissions Not Validated

**Location:** `cmd/client/config.go:27-30`

**Issue:** The config file is read without checking file permissions. A config file containing tokens could be world-readable.

**Risk:** LOW - This is a client-side CLI tool. Users are responsible for their own file permissions. The token can also be provided via environment variable which is the recommended approach for automation.

**Recommendation:** Consider adding a warning if config file permissions are too permissive (e.g., world-readable).

**Status:** ACCEPTABLE - Standard CLI behavior.

---

#### SEC-P8-002: Token Visible in Process Arguments

**Location:** `cmd/server/server.go:21-22`

**Issue:** Auth tokens can be passed via `--api-auth-token` and `--mcp-auth-token` flags, making them visible in process listings (`ps aux`).

**Risk:** LOW - Environment variables (which are also supported via config.Load()) are the recommended approach for secrets. CLI flags are provided for convenience in development/testing.

**Recommendation:** Document that environment variables should be used for production deployments.

**Status:** ACCEPTABLE - Standard CLI pattern. Environment variables are supported as primary method.

---

#### SEC-P8-003: VerifySSL Config Option Not Implemented

**Location:** `cmd/client/config.go:15`, `cmd/client/http.go:17-21`

**Issue:** The `VerifySSL` config option exists but is not used when creating the HTTP client.

**Risk:** LOW - Currently defaults to Go's secure defaults (TLS verification enabled). The option exists for future implementation.

**Recommendation:** Either implement the option or remove it to avoid confusion.

**Status:** ACCEPTABLE - Go defaults to secure TLS verification.

---

## Positive Security Observations

1. **Token not logged:** Auth tokens are not logged or printed in any output.

2. **Bearer token auth:** Proper Bearer token authentication header format used.

3. **Timeout configured:** HTTP client has configurable timeout preventing indefinite hangs.

4. **Environment variable support:** Tokens can be provided via `RACKD_TOKEN` environment variable, which is more secure than CLI flags.

5. **Error messages sanitized:** Error output doesn't leak sensitive information.

6. **XDG compliance:** Config directory follows XDG Base Directory specification.

---

## Recommendations Summary

| ID | Priority | Recommendation | Action |
|----|----------|----------------|--------|
| SEC-P8-001 | LOW | Warn on permissive config file permissions | Optional |
| SEC-P8-002 | LOW | Document env vars for production secrets | Documentation |
| SEC-P8-003 | LOW | Implement or remove VerifySSL option | Future task |

---

## Conclusion

The CLI implementation follows security best practices. All findings are low priority and represent standard CLI patterns. The code properly handles authentication tokens and doesn't expose sensitive data in logs or error messages.

**Review Result:** PASSED
