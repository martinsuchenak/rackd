# Security & Consistency Review

**Date:** 2026-03-04
**Review Scope:** Complete codebase review including API, MCP, CLI, Service Layer, Storage, Discovery, Web UI, and Tests
**Reviewers:** Security Reviewer, Code Reviewer (OMC Agents)

---

## Executive Summary

| Module | Critical | High | Medium | Low | Risk Level |
|--------|----------|------|--------|-----|------------|
| Authentication (`/internal/auth/`) | 1 | 4 | 5 | 2 | HIGH |
| API Handlers (`/internal/api/`) | 0 | 3 | 6 | 4 | MEDIUM |
| Storage Layer (`/internal/storage/`) | 0 | 3 | 5 | 4 | MEDIUM |
| Credentials (`/internal/credentials/`) | 0 | 2 | 3 | 2 | MEDIUM |
| MCP Server (`/internal/mcp/`) | 0 | 2 | 4 | 2 | MEDIUM |
| Service Layer (`/internal/service/`) | 2 | 4 | 6 | 3 | HIGH |
| CLI Commands (`/cmd/`) | 2 | 4 | 5 | 3 | HIGH |
| Discovery (`/internal/discovery/`) | 1 | 3 | 5 | 4 | HIGH |
| Web UI (`/webui/src/`) | 1 | 2 | 3 | 2 | HIGH |
| Tests | 0 | 1 | 5 | 4 | MEDIUM |
| **TOTAL** | **7** | **28** | **47** | **30** | **HIGH** |

---

## Critical Issues (Fix Immediately)

### 1. Session Token Not Invalidated on Password Change
**Module:** Authentication
**Category:** Broken Authentication
**Location:** `/internal/service/user.go:229,258`

**Issue:** When a user changes their password (via `ChangePassword` or `ResetPassword`), their existing sessions are NOT invalidated. This allows an attacker who obtained a session token before the password change to maintain access.

**Remediation:**
```go
// In ChangePassword (line 229) and ResetPassword (line 258), add:
s.sessions.InvalidateUserSessions(user.ID)
```

---

## High Issues

### 2. Missing SameSite=Strict on Session Cookies
**Module:** Authentication
**Category:** CSRF
**Location:** `/internal/api/auth_handlers.go:14-23`

**Issue:** Session cookies use `SameSiteLaxMode`. For authentication cookies, `SameSiteStrictMode` provides stronger CSRF protection.

**Remediation:** Change to `http.SameSiteStrictMode` or implement CSRF tokens.

---

### 3. In-Memory Session Store - No Persistence
**Module:** Authentication
**Category:** Availability
**Location:** `/internal/auth/session.go:26`

**Issue:** Sessions stored in memory with no persistence. Server restarts invalidate all sessions. Cannot scale horizontally.

**Remediation:** Consider Redis or database-backed session store for production.

---

### 4. OAuth Authorization Code Race Condition
**Module:** Authentication
**Category:** OAuth Implementation
**Location:** `/internal/service/oauth.go:214`

**Issue:** Authorization code lookup and marking as used are not atomic. Race condition allows potential code replay.

**Remediation:** Use database transactions or optimistic locking.

---

### 5. Legacy API Keys Bypass RBAC
**Module:** Authentication / Service Layer
**Category:** Broken Access Control
**Location:** `/internal/service/rbac.go:22-25`

**Issue:** Legacy API keys (without UserID) completely bypass all RBAC checks, granting unrestricted access.

**Remediation:** Migrate to user-associated keys or implement separate permission system.

---

### 6. Missing Permission Check in DNS LinkRecord Method
**Module:** Service Layer
**Category:** Broken Access Control
**Location:** `/internal/service/dns.go:1200-1272`

**Issue:** The `LinkRecord` method does not call `requirePermission()` before performing operations. Any authenticated user can link any DNS record to any device.

**Remediation:** Add permission check at the start of the method.

---

### 7. Insecure Password Input Handling in CLI
**Module:** CLI Commands
**Category:** Sensitive Data Exposure
**Location:** `/cmd/user/user.go:118-122,301-311`

**Issue:** Passwords are read using `fmt.Scanln()` which echoes input to the terminal, exposing passwords to shoulder surfing and terminal scrollback.

**Remediation:** Use `golang.org/x/term.ReadPassword()` to read passwords without echoing.

---

### 8. Potential Command Injection in OS Fingerprinting
**Module:** Discovery
**Category:** Command Injection
**Location:** `/internal/discovery/os_fingerprint.go:63-69`

**Issue:** The `measureTTL` function passes user-controllable IP address directly to `exec.Command` for the `ping` command without validation.

**Remediation:** Validate IP address format before passing to the command.

---

### 9. Open Redirect Vulnerability in Login Flow
**Module:** Web UI
**Category:** OWASP A01:2021 - Broken Access Control
**Location:** `/webui/src/components/login.ts:69-70`

**Issue:** The login component trusts the `redirect` query parameter without validation, allowing an attacker to craft malicious URLs that redirect users to external sites after login.

**Remediation:** Validate redirect parameter to only allow relative paths starting with `/`.

---

### 10. Missing ID Validation in Path Parameters
**Module:** API Handlers
**Category:** Input Validation
**Location:** Multiple handlers

**Files affected:**
- `/internal/api/device_handlers.go:52`
- `/internal/api/network_handlers.go:40`
- `/internal/api/user_handlers.go:30`
- `/internal/api/apikey_handlers.go:59`
- `/internal/api/dns_handlers.go:50,62,80`

**Remediation:** Add consistent ID validation:
```go
id := r.PathValue("id")
if id == "" {
    h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
    return
}
```

---

### 11. API Keys Stored in Plaintext
**Module:** MCP Server / Storage
**Category:** Sensitive Data Exposure
**Location:** `/internal/storage/apikey_sqlite.go:32-43`

**Issue:** API keys are stored in plaintext in the database. If the database is compromised, all API keys are immediately usable by attackers.

**Remediation:** Hash API keys before storage using SHA-256, similar to OAuth tokens.

---

### 12. No Rate Limiting on MCP Endpoint
**Module:** MCP Server
**Category:** Denial of Service
**Location:** `/internal/server/server.go:123,248`

**Issue:** The MCP endpoint (`POST /mcp`) is registered directly without rate limiting middleware, allowing brute-force attacks on API keys.

**Remediation:** Apply rate limiting to the MCP endpoint.

---

### 13. Race Condition in IP Address Allocation
**Module:** Service Layer
**Category:** Race Condition / Data Integrity
**Location:** `/internal/service/reservation.go:86-96`

**Issue:** The `GetNextAvailableIP` call and subsequent reservation creation are not atomic, allowing the same IP to be reserved twice.

**Remediation:** Use database-level transactions or optimistic locking.

---

### 14. Token Passed via Command Line Flag
**Module:** CLI Commands
**Category:** Sensitive Data Exposure
**Location:** `/cmd/dns/provider.go:130,192`

**Issue:** DNS provider tokens can be passed via `--token` flag, visible in process listings and shell history.

**Remediation:** Accept tokens via environment variables or files only.

---

### 15. SNMPv2c Community String Transmitted in Cleartext
**Module:** Discovery
**Category:** Sensitive Data Exposure
**Location:** `/internal/discovery/snmp.go:61-65`

**Issue:** SNMPv2c transmits community strings in cleartext over the network. This is a significant security risk for production environments.

**Remediation:** Add configuration flag to disable SNMPv2c in production.

---

### 16. Potential XSS via Extension Pages
**Module:** Web UI
**Category:** OWASP A03:2021 - Injection (XSS)
**Location:** `/webui/src/index.html:366`

**Issue:** Extension pages use `x-html` to inject arbitrary HTML content. If an extension is compromised or malicious, it could inject XSS payloads.

**Remediation:** Implement CSP and consider using DOMPurify for sanitization.

---

### 17. Missing Input Validation in Bulk Operations
**Module:** API Handlers
**Category:** Input Validation
**Location:** `/internal/api/handlers.go:224-306`

**Issue:** Bulk operations accept arrays without:
- Array size limits (DoS vector)
- Individual item validation
- Required field checks

**Remediation:**
```go
if len(devices) > 100 {
    h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Maximum 100 items")
    return
}
```

---

### 8. Missing Rate Limiting on Sensitive Endpoints
**Module:** API Handlers
**Category:** Rate Limiting
**Location:** `/internal/api/handlers.go:69-325`

**Issue:** Only login has rate limiting. Missing on:
- Password reset
- User creation
- API key creation
- OAuth token endpoint
- Bulk operations

**Remediation:** Apply rate limiting to all sensitive endpoints.

---

### 9. Missing Context Propagation in Storage
**Module:** Storage Layer
**Category:** Context Usage
**Location:** Multiple files

**Files affected:**
- `/internal/storage/device_sqlite.go:21`
- `/internal/storage/user_sqlite.go:64`
- `/internal/storage/audit_sqlite.go:20`

**Issue:** Methods use `context.Background()` or no context at all, preventing timeout propagation.

**Remediation:** Update interface to accept `context.Context` as first parameter.

---

### 10. Goroutine Leak in Audit Logging
**Module:** Storage Layer
**Category:** Resource Management
**Location:** `/internal/storage/sqlite.go:166-196`

**Issue:** Audit logging spawns goroutines without limit. Context may be canceled before completion. Fire-and-forget pattern silently ignores failures.

**Remediation:** Use buffered channel with fixed worker pool.

---

### 11. LIKE Query Pattern in Webhook Event Matching
**Module:** Storage Layer
**Category:** Query Efficiency
**Location:** `/internal/storage/webhook_sqlite.go:141-146`

**Issue:** LIKE with wildcards matches unintended events (`device` matches `device_created`, etc.).

**Remediation:** Use JSON query functions or normalized table structure.

---

### 12. Silent Decryption Failures in Database Reads
**Module:** Credentials
**Category:** Error Handling
**Location:** `/internal/credentials/storage.go:196-200,214-218`

**Issue:** Decryption errors silently ignored. Corrupted/tampered data goes undetected.

```go
cred.SNMPCommunity, _ = s.encryptor.Decrypt(community.String)  // Error ignored!
```

**Remediation:** Return decryption errors to caller.

---

### 13. Missing Key Rotation Implementation
**Module:** Credentials
**Category:** Key Management
**Location:** Documentation claims feature exists but not implemented

**Issue:** No mechanism to rotate encryption keys. Documentation references `--rotate-keys` flag that doesn't exist.

**Remediation:** Implement key rotation or remove documentation.

---

## Medium Issues

### Authentication Module

| # | Issue | Location |
|---|-------|----------|
| 14 | No rate limiting on OAuth token endpoint | `/internal/api/oauth_handlers.go:170` |
| 15 | Weak password length (8 chars minimum) | `/internal/service/user.go:58-59` |
| 16 | bcrypt cost factor could be higher | `/internal/auth/password.go:10` |
| 17 | No refresh token rotation | `/internal/service/oauth.go:265` |
| 18 | Wildcard scope (*) grants full access | `/internal/auth/oauth.go:74-75` |

### API Handlers Module

| # | Issue | Location |
|---|-------|----------|
| 19 | Missing CSRF protection for session auth | `/internal/api/middleware.go` |
| 20 | CSP allows 'unsafe-eval' and 'unsafe-inline' | `/internal/api/middleware.go:218-222` |
| 21 | Potential info leakage in error responses | `/internal/api/handlers.go:342-371` |
| 22 | Missing CORS configuration | Entire API layer |
| 23 | No request timeout enforcement | `/internal/api/handlers.go` |
| 24 | Localhost bypass in rate limiting | `/internal/api/ratelimit.go:132-136` |

### Storage Layer Module

| # | Issue | Location |
|---|-------|----------|
| 25 | Missing pagination limits on list operations | Multiple files |
| 26 | Insufficient IP address validation | `/internal/storage/pool_sqlite.go:319-333` |
| 27 | JSON unmarshal errors ignored | Multiple files |
| 28 | Missing rate limiting for OAuth operations | `/internal/storage/oauth_sqlite.go` |
| 29 | Bulk operations continue on partial failure | `/internal/storage/bulk.go:28-35` |

### Credentials Module

| # | Issue | Location |
|---|-------|----------|
| 30 | No secure memory handling for keys | `/internal/credentials/encrypt.go:14-16` |
| 31 | Development mode warning insufficient | `/internal/cmd/server/server.go:97-102` |
| 32 | No nonce reuse detection beyond GCM guarantees | `/internal/credentials/encrypt.go:37-41` |

### MCP Server Module

| # | Issue | Location |
|---|-------|----------|
| 33 | Legacy API keys bypass RBAC | `/internal/service/rbac.go:22-25` |
| 34 | No MCP-specific audit logging | `/internal/mcp/server.go` |
| 35 | Self-relationship not prevented | `/internal/mcp/server.go:409-422` |
| 36 | Missing input validation for IP/CIDR | `/internal/mcp/server.go:498-522` |

### Service Layer Module

| # | Issue | Location |
|---|-------|----------|
| 37 | Self-privilege escalation via user update | `/internal/service/user.go:126-169` |
| 38 | OAuth client credentials scope validation missing | `/internal/service/oauth.go:344-347` |
| 39 | No rate limiting for authentication attempts | `/internal/service/auth.go:31-69` |
| 40 | Webhook SSRF protection incomplete | `/internal/service/webhook.go:304-337` |
| 41 | OAuth authorization code replay window | `/internal/service/oauth.go:205-216` |
| 42 | Missing input validation for IP addresses | `/internal/service/device.go:222-253` |

### CLI Commands Module

| # | Issue | Location |
|---|-------|----------|
| 43 | No SSL verification control | `/cmd/client/config.go:15` |
| 44 | Webhook secret passed via command line | `/cmd/webhook/create.go:22` |
| 45 | Default HTTP server URL | `/cmd/client/config.go:18-19` |
| 46 | No rate limiting on API requests | `/cmd/client/http.go` |
| 47 | API key printed to terminal | `/cmd/apikey/apikey.go:132` |

### Discovery Module

| # | Issue | Location |
|---|-------|----------|
| 48 | Credential decryption errors silently ignored | `/internal/credentials/storage.go:196-200` |
| 49 | No rate limiting for network scans | `/internal/discovery/unified_scanner.go:254` |
| 50 | Unbounded subnet size (up to /16) | `/internal/discovery/helpers.go:8-11` |
| 51 | Sensitive data may be logged | `/internal/discovery/unified_scanner.go:291-295` |
| 52 | Memory exhaustion from result accumulation | `/internal/discovery/unified_scanner.go:250-251` |

### Web UI Module

| # | Issue | Location |
|---|-------|----------|
| 53 | Sensitive data in client-side memory | `/webui/src/components/credentials.ts:22-28` |
| 54 | Missing CSRF token implementation | `/webui/src/core/api.ts` |
| 55 | Weak email validation | `/webui/src/components/users.ts:381-382` |

### Tests Module

| # | Issue | Location |
|---|-------|----------|
| 56 | Hardcoded test API key pattern | `/internal/api/integration_test.go:47` |
| 57 | Hardcoded password in bootstrap tests | `/internal/storage/bootstrap_test.go:22,59` |
| 58 | Hardcoded secrets in webhook tests | `/internal/storage/webhook_sqlite_test.go:22` |
| 59 | Localhost bypass in rate limiting tests | `/internal/api/ratelimit_test.go:98-119` |
| 60 | OAuth redirect URI uses localhost | `/internal/storage/oauth_sqlite_test.go:17` |

---

## Low Issues

| # | Module | Issue | Location |
|---|--------|-------|----------|
| 61 | Auth | Generic error messages could leak timing info | `/internal/service/auth.go:31-38` |
| 62 | Auth | No CSP nonce implementation | `/internal/api/middleware.go:222` |
| 63 | API | Test API key in integration test | `/internal/api/integration_test.go:47` |
| 64 | API | Missing input validation for query params | Multiple handlers |
| 65 | API | Webhook secret not validated on creation | `/internal/api/webhook_handlers.go:58-73` |
| 66 | API | Missing validation in network update handler | `/internal/api/network_handlers.go:50-86` |
| 67 | Storage | Inconsistent error types | Multiple files |
| 68 | Storage | Database file permissions not explicitly set | `/internal/storage/sqlite.go:32-35` |
| 69 | Storage | No row-level security or tenant isolation | All storage files |
| 70 | Storage | Connection pool limited to single connection | `/internal/storage/sqlite.go:45-47` |
| 71 | Credentials | Empty plaintext returns empty string | `/internal/credentials/encrypt.go:34-36` |
| 72 | Credentials | No encryption algorithm versioning | `/internal/credentials/encrypt.go:33-42` |
| 73 | Service | Admin check error ignored | `/internal/service/apikey.go:120` |
| 74 | Service | Silent failure in DNS sync | `/internal/service/device.go:247-250` |
| 75 | Service | Potential integer overflow in port validation | `/internal/service/nat.go:67-72` |
| 76 | CLI | Password comparison timing attack | `/cmd/user/user.go:124` |
| 77 | CLI | Error messages may contain sensitive info | `/cmd/client/errors.go:28-30` |
| 78 | CLI | No request timeout enforcement | `/cmd/client/http.go:20` |
| 79 | Discovery | Missing context timeout in SSH commands | `/internal/discovery/ssh.go:110-118` |
| 80 | Discovery | Hardcoded port numbers | Multiple files |
| 81 | Discovery | No validation of NetBIOS hostname chars | `/internal/discovery/unified_scanner.go:427-436` |
| 82 | Discovery | Result cache without size limit | `/internal/discovery/adaptive.go:273-281` |
| 83 | Web UI | No explicit CSP | `/webui/src/index.html` |
| 84 | Web UI | Password policy client-side only | `/webui/src/components/login.ts:39-42` |
| 85 | Tests | Test tokens not using secure random | `/internal/auth/apikey_test.go:14` |
| 86 | Tests | Password test uses short password | `/internal/auth/password_test.go:22` |
| 87 | Tests | No security tests for injection attacks | N/A - Missing |
| 88 | Tests | HTTP used in test data script | `/testdata/load-testdata.sh:10,34` |

---

## Consistency Issues

### API Handlers

| Issue | Severity | Files Affected |
|-------|----------|----------------|
| Inconsistent JSON decode error codes (`INVALID_JSON` vs `INVALID_INPUT`) | HIGH | Multiple |
| Inconsistent update patterns (map vs struct-based) | HIGH | Multiple |
| Duplicate code in update handlers (type assertions) | HIGH | device_handlers.go, network_handlers.go, datacenter_handlers.go |
| Inconsistent error response format for missing fields | HIGH | Multiple |
| Inconsistent GET vs POST responses for delete operations | HIGH | Multiple |
| Inconsistent null array handling | MEDIUM | Multiple |
| No pagination support on many list endpoints | MEDIUM | Multiple |
| Inconsistent logging patterns | MEDIUM | Multiple |
| Inconsistent request struct definitions (local vs model package) | MEDIUM | Multiple |
| Validation inconsistency (handler vs service layer) | MEDIUM | Multiple |
| Health endpoint uses different response pattern | MEDIUM | health_handlers.go |
| Inconsistent action endpoint responses | MEDIUM | Multiple |
| Inconsistent error message style (periods) | LOW | Multiple |
| Inline struct vs named struct for simple requests | LOW | Multiple |

### Storage Layer

| Issue | Severity | Files Affected |
|-------|----------|----------------|
| Inconsistent "not found" error handling (sentinel vs inline) | HIGH | user_sqlite.go, rbac_sqlite.go, conflict_sqlite.go |
| Inconsistent transaction usage for write operations | HIGH | reservation_sqlite.go, discovery_sqlite.go, user_sqlite.go |
| Mixed context creation patterns | HIGH | Multiple files |
| Duplicate nullTime helper function | MEDIUM | conflict_sqlite.go |
| Inconsistent NULL string handling | MEDIUM | webhook_sqlite.go, oauth_sqlite.go |
| Mixed time.Now().UTC() vs time.Now() usage | MEDIUM | Multiple files |
| Mixed UUID generation approaches | MEDIUM | oauth_sqlite.go, audit_sqlite.go |
| Inconsistent empty slice initialization | MEDIUM | Multiple files |
| Inconsistent audit logging on write operations | MEDIUM | user_sqlite.go, rbac_sqlite.go, oauth_sqlite.go |
| Inconsistent nil pointer validation | MEDIUM | Multiple files |
| Repeated scan helper patterns | LOW | Multiple files |
| Interface definitions in wrong files | MEDIUM | user_sqlite.go, rbac_sqlite.go |

---

## Positive Security Findings

### Authentication
- Strong password hashing (bcrypt cost 12)
- Constant-time API key comparison
- PKCE enforcement for OAuth public clients (S256)
- OAuth tokens stored as SHA-256 hashes
- Secure cookie flags (HttpOnly, Secure, SameSite)
- Login rate limiting implemented
- Comprehensive security headers

### API Handlers
- All database queries use parameterized queries
- RBAC implementation in service layer
- Request body size limited (1MB)
- Webhook secret sanitization in responses
- Legacy API key bypass is documented

### Storage Layer
- Foreign key constraints enabled
- Input validation for empty IDs
- Secrets stored as hashes
- Audit logging infrastructure

### Credentials
- Correct AES-256-GCM implementation
- Proper nonce generation (crypto/rand)
- Key validation (32 bytes)
- Non-deterministic encryption (fresh nonce each time)
- Tests cover key scenarios

---

## Recommended Action Plan

### Immediate (This Week)
1. **Fix session invalidation on password change** (Critical)
2. Add rate limiting to OAuth token endpoint
3. Fix silent decryption errors in credentials module

### Short Term (This Month)
1. Change session cookie to SameSite=Strict
2. Add bulk operation size limits
3. Add ID validation to all path parameters
4. Standardize error codes across API handlers
5. Fix authorization code race condition

### Medium Term (Next Quarter)
1. Implement persistent session store
2. Implement refresh token rotation
3. Add encryption key rotation capability
4. Migrate legacy API keys to user-associated keys
5. Standardize storage layer context usage

### Long Term (Backlog)
1. Implement CSP nonces
2. Add request timeout middleware
3. Implement CORS configuration
4. Add secure memory handling for encryption keys
5. Standardize all request structs in model package

---

## Files Requiring Immediate Attention

| File | Issues | Priority |
|------|--------|----------|
| `/internal/service/user.go` | Session invalidation on password change | CRITICAL |
| `/internal/credentials/storage.go` | Silent decryption errors | HIGH |
| `/internal/api/handlers.go` | Bulk operation validation, rate limiting | HIGH |
| `/internal/service/oauth.go` | Authorization code race, token rotation | HIGH |
| `/internal/auth/session.go` | Persistence, scalability | HIGH |

---

*Report generated by OMC Security Review Agents*

---

## Additional Module Findings (Extended Review)

### MCP Server
**Additional Issues Found:**
- API keys stored in plaintext (HIGH)
- No rate limiting on MCP endpoint (HIGH)
- No MCP-specific audit logging (MEDIUM)
- Self-relationship not prevented (MEDIUM)
- Missing input validation for IP/CIDR (MEDIUM)

**Positive Findings:**
- OAuth 2.1 compliance with PKCE
- Token hashing with SHA-256
- Constant-time comparison for client secrets
- Proper WWW-Authenticate headers

### Service Layer
**Additional Issues Found:**
- Missing permission check in DNS LinkRecord (CRITICAL)
- Missing permission check in DNS PromoteRecord (HIGH)
- Self-privilege escalation via user update (HIGH)
- Race condition in IP address allocation (HIGH)
- OAuth client credentials scope validation missing (HIGH)

**Positive Findings:**
- Consistent RBAC pattern with requirePermission()
- Proper bcrypt password hashing
- HMAC-SHA256 for webhook signatures
- Audit context propagation

### CLI Commands
**Additional Issues Found:**
- Insecure password input handling - echo to terminal (CRITICAL)
- Password length validation too weak (CRITICAL)
- No URL validation for webhook URLs - SSRF risk (HIGH)
- Token passed via command line flag (HIGH)
- API key printed to terminal (HIGH)

**Positive Findings:**
- No hardcoded secrets (uses environment variables)
- Bearer token authentication
- No command injection patterns

### Discovery Module
**Additional Issues Found:**
- Potential command injection in OS fingerprinting (CRITICAL)
- Missing input validation for IP addresses in ARP scanner (HIGH)
- SNMPv2c community string transmitted in cleartext (HIGH)
- Trust-On-First-Use (TOFU) for SSH host keys (HIGH)
- No rate limiting for network scans (MEDIUM)

**Positive Findings:**
- Credential encryption at rest
- Context cancellation support
- Subnet size limit (/16 max)
- SNMPv3 support with SHA/AES

### Web UI
**Additional Issues Found:**
- Open redirect vulnerability in login flow (CRITICAL)
- Potential XSS via extension pages (HIGH)
- OAuth redirect URI trust issue (HIGH)
- Sensitive data in client-side memory (MEDIUM)
- Missing CSRF token implementation (MEDIUM)

**Positive Findings:**
- No hardcoded secrets
- credentials: same-origin used consistently
- No dynamic code execution
- Password-type inputs for sensitive fields

### Tests
**Additional Issues Found:**
- Hardcoded test API key pattern (HIGH)
- Hardcoded password in bootstrap tests (MEDIUM)
- Hardcoded secrets in webhook tests (MEDIUM)
- Localhost bypass in rate limiting tests (MEDIUM)
- No security tests for injection attacks (LOW)

**Positive Findings:**
- No real secrets found (all test values)
- Good encryption test coverage
- Authentication tests present
- Rate limiting tests present

---

## Coverage Summary

| Component | Reviewed | Issues Found |
|-----------|----------|--------------|
| /internal/auth/ | Yes | 12 |
| /internal/api/ | Yes | 13 |
| /internal/storage/ | Yes | 12 |
| /internal/credentials/ | Yes | 5 |
| /internal/mcp/ | Yes | 8 |
| /internal/service/ | Yes | 15 |
| /internal/discovery/ | Yes | 13 |
| /cmd/ | Yes | 14 |
| /webui/src/ | Yes | 8 |
| Tests | Yes | 10 |

---

## Updated Action Plan

### Immediate (This Week) - CRITICAL
1. Fix session invalidation on password change
2. Add permission checks to DNS LinkRecord and PromoteRecord
3. Fix CLI password input to use term.ReadPassword
4. Fix open redirect vulnerability in Web UI login
5. Add IP validation in OS fingerprinting
6. Migrate legacy API keys or remove bypass

### Short Term (This Month) - HIGH
1. Hash API keys before storage
2. Add rate limiting to MCP endpoint
3. Add bulk operation size limits
4. Add SSRF protection for webhook URLs
5. Fix authorization code race condition
6. Add configuration to disable SNMPv2c in production

### Medium Term (Next Quarter)
1. Implement persistent session store
2. Implement refresh token rotation
3. Add encryption key rotation capability
4. Standardize storage layer context usage
5. Add input validation tests for injection attacks
6. Implement CSP for Web UI

---

*Extended review completed with full codebase coverage*
