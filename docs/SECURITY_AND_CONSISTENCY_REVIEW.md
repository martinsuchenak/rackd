# Security & Consistency Review

**Date:** 2026-03-04
**Review Scope:** Complete codebase review including API, MCP, CLI, Service Layer, Storage, Discovery, Web UI, and Tests
**Reviewers:** Security Reviewer, Code Reviewer (OMC Agents)
**Last Updated:** 2026-03-04

---

## Executive Summary

| Module | Critical | High | Medium | Low | Risk Level |
|--------|----------|------|--------|-----|------------|
| Authentication (`/internal/auth/`) | 1 | 2 | 5 | 2 | HIGH |
| API Handlers (`/internal/api/`) | 0 | 3 | 6 | 4 | MEDIUM |
| Storage Layer (`/internal/storage/`) | 0 | 3 | 5 | 4 | MEDIUM |
| Credentials (`/internal/credentials/`) | 0 | 2 | 3 | 2 | MEDIUM |
| MCP Server (`/internal/mcp/`) | 0 | 2 | 4 | 2 | MEDIUM |
| Service Layer (`/internal/service/`) | 2 | 3 | 6 | 3 | HIGH |
| CLI Commands (`/cmd/`) | 2 | 3 | 5 | 3 | HIGH |
| Discovery (`/internal/discovery/`) | 1 | 2 | 5 | 4 | HIGH |
| Web UI (`/webui/src/`) | 1 | 2 | 3 | 2 | HIGH |
| Tests | 0 | 1 | 5 | 4 | MEDIUM |
| **TOTAL** | **7** | **23** | **47** | **30** | **HIGH** |

---

## 1. ✅ Completed Issues (Fixed)

**Critical**
1. **Session Token Not Invalidated on Password Change**
   - **Module:** Authentication
   - **Fix Applied:** Added session invalidation to both `ChangePassword` and `ResetPassword` methods in `/internal/service/user.go`. All existing sessions are now invalidated when a password is changed.
2. **Missing Permission Check in DNS LinkRecord Method**
   - **Module:** Service Layer
   - **Fix Applied:** Added `requirePermission(ctx, s.store, "dns", "update")` to both `LinkRecord` and `PromoteRecord` methods.
3. **Insecure Password Input Handling in CLI**
   - **Module:** CLI Commands
   - **Fix Applied:** Replaced `fmt.Scanln()` with `term.ReadPassword()` in both `CreateCommand` and `ChangePasswordCommand`. Passwords are no longer echoed to the terminal.
4. **Potential Command Injection in OS Fingerprinting**
   - **Module:** Discovery
   - **Fix Applied:** Added defense-in-depth IP validation inside `measureTTL()` using `net.ParseIP()`.
5. **Open Redirect Vulnerability in Login Flow**
   - **Module:** Web UI
   - **Fix Applied:** Added validation to only allow relative paths starting with `/` and rejecting URLs that start with `//`. Invalid redirects are replaced with `/`.
6. **Legacy API Keys Bypass RBAC**
   - **Module:** Authentication / Service Layer
   - **Fix Applied:** Removed the legacy API key bypass entirely. Legacy API keys (without user association) are now rejected.

**High**
7. **Missing SameSite=Strict on Session Cookies**
   - **Module:** Authentication
   - **Fix Applied:** Changed to `http.SameSiteStrictMode` for session cookies in `/internal/api/auth_handlers.go`.
8. **Missing ID Validation in Path Parameters**
   - **Module:** API Handlers
   - **Fix Applied:** Added consistent ID validation across multiple handlers.
9. **API Keys Stored in Plaintext**
   - **Module:** MCP Server / Storage
   - **Fix Applied:** Hashed API keys before storage and stored SHA-256 hashes instead of plaintext tokens.
10. **No Rate Limiting on MCP Endpoint**
    - **Module:** MCP Server
    - **Fix Applied:** Applied rate limiting wrapper to the MCP endpoint `POST /mcp` in `/internal/server/server.go`.
11. **Missing Input Validation in Bulk Operations**
    - **Module:** API Handlers
    - **Fix Applied:** Added array size limits for all bulk operations in `device_handlers.go` and `network_handlers.go`.
12. **Missing Rate Limiting on Sensitive Endpoints**
    - **Module:** API Handlers
	- **Fix Applied:** Introduced `wrapSensitiveAuth` and `wrapSensitiveNoAuth` to apply rate limits to password resets, user creation, API key creation, OAuth tokens, and bulk operations endpoints.
13. **In-Memory Session Store - No Persistence**
    - **Module:** Authentication
    - **Fix Applied:** Implemented a persistent session store using SQLite by default, with an option to use a Valkey/Redis store instead. Added `SESSION_STORE_TYPE` and `VALKEY_URL` configuration variables.
14. **OAuth Authorization Code Race Condition**
    - **Module:** Authentication
    - **Fix Applied:** Modified `MarkAuthorizationCodeUsed` in the SQLite storage layer to update the used flag only if it is currently 0 (`AND used = 0`) and fail with `ErrOAuthCodeUsed` if no rows were affected. This guarantees atomic marking and prevents race conditions that could lead to authorization code replay.
15. **Race Condition in IP Address Allocation**
    - **Module:** Service Layer
    - **Fix Applied:** Updated `CreateReservation` in SQLite storage to proactively transform SQLite Unique Constraint violations into `ErrIPAlreadyReserved` when identical `(pool_id, ip_address)` reservations clash. Updated `service/reservation.go` to intercept these conflicts and retry allocation dynamically up to 5 times.
16. **Token Passed via Command Line Flag**
    - **Module:** CLI Commands 
    - **Fix Applied:** Replaced the `--token` flag with `--token-env` and `--token-file` in the `dns provider create` and `dns provider update` subcommands, precluding the possibility of API credentials lingering in process namespaces or shell history.
17. **SNMPv2c Community String Transmitted in Cleartext**
    - **Module:** Discovery
    - **Fix Applied:** Introduced a `DISCOVERY_SNMPV2C_ENABLED` configuration parameter. SNMPv2c is now disabled by default and gracefully handles unpermitted scans. Administrator must explicitly flip to `true` to use cleartext device discovery scanning.

---

## 2. 🔴 Open Critical Issues
*(All identified critical issues have been fixed)*

---

## 3. 🟠 Open High Issues



### H-6: Potential XSS via Extension Pages
**Module:** Web UI
**Category:** Injection (XSS)
**Location:** `/webui/src/index.html:366`
**Issue:** Extension pages use `x-html` to inject arbitrary HTML content. If an extension is compromised, it could inject XSS payloads.
**Remediation:** Verify if there are any extensions in use. If not, remove the extension functionality. If yes, implement CSP and consider using DOMPurify for sanitization.

### H-7: Missing Context Propagation in Storage
**Module:** Storage Layer
**Category:** Context Usage
**Location:** Multiple files (e.g., `device_sqlite.go:21`, `user_sqlite.go:64`)
**Issue:** Methods use `context.Background()` or no context at all, preventing timeout propagation.
**Remediation:** Update interface to accept `context.Context` as first parameter.

### H-8: Goroutine Leak in Audit Logging
**Module:** Storage Layer
**Category:** Resource Management
**Location:** `/internal/storage/sqlite.go:166-196`
**Issue:** Audit logging spawns goroutines without limit. Fire-and-forget pattern silently ignores failures.
**Remediation:** Use buffered channel with fixed worker pool.

### H-9: LIKE Query Pattern in Webhook Event Matching
**Module:** Storage Layer
**Category:** Query Efficiency
**Location:** `/internal/storage/webhook_sqlite.go:141-146`
**Issue:** LIKE with wildcards matches unintended events (`device` matches `device_created`, etc.).
**Remediation:** Use JSON query functions or normalized table structure.

### H-10: Silent Decryption Failures in Database Reads
**Module:** Credentials
**Category:** Error Handling
**Location:** `/internal/credentials/storage.go:196-200,214-218`
**Issue:** Decryption errors silently ignored. Corrupted/tampered data goes undetected.
**Remediation:** Return decryption errors to caller.

### H-11: Missing Key Rotation Implementation
**Module:** Credentials
**Category:** Key Management
**Location:** Architecture / CLI logic
**Issue:** No mechanism to rotate encryption keys.
**Remediation:** Implement key rotation.

### H-12: Additional High Issues from Extended Review
- **Discovery:** Missing input validation for IP addresses in ARP scanner. Trust-On-First-Use (TOFU) issue for SSH host keys.
- **Service Layer / API:** Self-privilege escalation via user update. OAuth client credentials scope validation missing.
- **CLI / Service:** No URL validation for webhook URLs (SSRF risk). API key printed to terminal output.
- **Web UI & Tests:** OAuth redirect URI trust issue. Hardcoded test API key patterns.

---

## 🟡 Open Medium Issues

### Authentication Module
| # | Issue | Location |
|---|-------|----------|
| M-1 | No rate limiting on OAuth token endpoint | `/internal/api/oauth_handlers.go:170` |
| M-2 | Weak password length (8 chars minimum) | `/internal/service/user.go:58-59` |
| M-3 | bcrypt cost factor could be higher | `/internal/auth/password.go:10` |
| M-4 | No refresh token rotation | `/internal/service/oauth.go:265` |
| M-5 | Wildcard scope (*) grants full access | `/internal/auth/oauth.go:74-75` |

### API Handlers Module
| # | Issue | Location |
|---|-------|----------|
| M-6 | Missing CSRF protection for session auth | `/internal/api/middleware.go` |
| M-7 | CSP allows 'unsafe-eval' and 'unsafe-inline' | `/internal/api/middleware.go:218-222` |
| M-8 | Potential info leakage in error responses | `/internal/api/handlers.go:342-371` |
| M-9 | Missing CORS configuration | Entire API layer |
| M-10 | No request timeout enforcement | `/internal/api/handlers.go` |
| M-11 | Localhost bypass in rate limiting | `/internal/api/ratelimit.go:132-136` |

### Storage Layer Module
| # | Issue | Location |
|---|-------|----------|
| M-12 | Missing pagination limits on list operations | Multiple files |
| M-13 | Insufficient IP address validation | `/internal/storage/pool_sqlite.go:319-333` |
| M-14 | JSON unmarshal errors ignored | Multiple files |
| M-15 | Missing rate limiting for OAuth operations | `/internal/storage/oauth_sqlite.go` |
| M-16 | Bulk operations continue on partial failure | `/internal/storage/bulk.go:28-35` |

### Credentials Module
| # | Issue | Location |
|---|-------|----------|
| M-17 | No secure memory handling for keys | `/internal/credentials/encrypt.go:14-16` |
| M-18 | Development mode warning insufficient | `/internal/cmd/server/server.go:97-102` |
| M-19 | No nonce reuse detection beyond GCM guarantees | `/internal/credentials/encrypt.go:37-41` |

### MCP Server Module
| # | Issue | Location |
|---|-------|----------|
| M-20 | No MCP-specific audit logging | `/internal/mcp/server.go` |
| M-21 | Self-relationship not prevented | `/internal/mcp/server.go:409-422` |
| M-22 | Missing input validation for IP/CIDR | `/internal/mcp/server.go:498-522` |

### Service Layer Module
| # | Issue | Location |
|---|-------|----------|
| M-23 | No rate limiting for authentication attempts | `/internal/service/auth.go:31-69` |
| M-24 | Webhook SSRF protection incomplete | `/internal/service/webhook.go:304-337` |
| M-25 | OAuth authorization code replay window | `/internal/service/oauth.go:205-216` |
| M-26 | Missing input validation for IP addresses | `/internal/service/device.go:222-253` |

### CLI Commands Module
| # | Issue | Location |
|---|-------|----------|
| M-27 | No SSL verification control | `/cmd/client/config.go:15` |
| M-28 | Webhook secret passed via command line | `/cmd/webhook/create.go:22` |
| M-29 | Default HTTP server URL | `/cmd/client/config.go:18-19` |
| M-30 | No rate limiting on API requests | `/cmd/client/http.go` |

### Discovery Module
| # | Issue | Location |
|---|-------|----------|
| M-31 | No rate limiting for network scans | `/internal/discovery/unified_scanner.go:254` |
| M-32 | Unbounded subnet size (up to /16) | `/internal/discovery/helpers.go:8-11` |
| M-33 | Sensitive data may be logged | `/internal/discovery/unified_scanner.go:291-295` |
| M-34 | Memory exhaustion from result accumulation | `/internal/discovery/unified_scanner.go:250-251` |

### Web UI Module
| # | Issue | Location |
|---|-------|----------|
| M-35 | Sensitive data in client-side memory | `/webui/src/components/credentials.ts:22-28` |
| M-36 | Missing CSRF token implementation | `/webui/src/core/api.ts` |
| M-37 | Weak email validation | `/webui/src/components/users.ts:381-382` |

### Tests Module
| # | Issue | Location |
|---|-------|----------|
| M-38 | Hardcoded password in bootstrap tests | `/internal/storage/bootstrap_test.go:22,59` |
| M-39 | Hardcoded secrets in webhook tests | `/internal/storage/webhook_sqlite_test.go:22` |
| M-40 | Localhost bypass in rate limiting tests | `/internal/api/ratelimit_test.go:98-119` |

---

## 🟢 Open Low Issues

| # | Module | Issue | Location |
|---|--------|-------|----------|
| L-1 | Auth | Generic error messages could leak timing info | `/internal/service/auth.go:31-38` |
| L-2 | Auth | No CSP nonce implementation | `/internal/api/middleware.go:222` |
| L-3 | API | Missing input validation for query params | Multiple handlers |
| L-4 | API | Webhook secret not validated on creation | `/internal/api/webhook_handlers.go:58-73` |
| L-5 | API | Missing validation in network update handler | `/internal/api/network_handlers.go:50-86` |
| L-6 | Storage | Inconsistent error types | Multiple files |
| L-7 | Storage | Database file permissions not explicitly set | `/internal/storage/sqlite.go:32-35` |
| L-8 | Storage | No row-level security or tenant isolation | All storage files |
| L-9 | Storage | Connection pool limited to single connection | `/internal/storage/sqlite.go:45-47` |
| L-10 | Credentials | Empty plaintext returns empty string | `/internal/credentials/encrypt.go:34-36` |
| L-11 | Credentials | No encryption algorithm versioning | `/internal/credentials/encrypt.go:33-42` |
| L-12 | Service | Admin check error ignored | `/internal/service/apikey.go:120` |
| L-13 | Service | Silent failure in DNS sync | `/internal/service/device.go:247-250` |
| L-14 | Service | Potential integer overflow in port validation | `/internal/service/nat.go:67-72` |
| L-15 | CLI | Password comparison timing attack | `/cmd/user/user.go:124` |
| L-16 | CLI | Error messages may contain sensitive info | `/cmd/client/errors.go:28-30` |
| L-17 | CLI | No request timeout enforcement | `/cmd/client/http.go:20` |
| L-18 | Discovery | Missing context timeout in SSH commands | `/internal/discovery/ssh.go:110-118` |
| L-19 | Discovery | Hardcoded port numbers | Multiple files |
| L-20 | Discovery | No validation of NetBIOS hostname chars | `/internal/discovery/unified_scanner.go:427-436` |
| L-21 | Discovery | Result cache without size limit | `/internal/discovery/adaptive.go:273-281` |
| L-22 | Web UI | No explicit CSP | `/webui/src/index.html` |
| L-23 | Web UI | Password policy client-side only | `/webui/src/components/login.ts:39-42` |
| L-24 | Tests | Test tokens not using secure random | `/internal/auth/apikey_test.go:14` |
| L-25 | Tests | Password test uses short password | `/internal/auth/password_test.go:22` |
| L-26 | Tests | No security tests for injection attacks | N/A - Missing |
| L-27 | Tests | HTTP used in test data script | `/testdata/load-testdata.sh:10,34` |

---

## 🔵 Consistency Issues

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

- **Authentication:** Strong password hashing (bcrypt cost 12), constant-time API key comparison, PKCE enforcement for OAuth, Hash-based OAuth storage, Secure cookies, Rate limiting logic in place.
- **API Handlers:** Parameterized queries globally, RBAC implemented correctly, Body size caps (1MB).
- **Storage Layer:** FK constraints enabled, ID validations, audit logging infrastructure.
- **Credentials:** Proper AES-256-GCM usage, good nonce generation, validation, and testing.

---

## Updated Action Plan

### Immediate (This Week) - CRITICAL
*(All critical issues documented in this review have been successfully resolved)*

### Short Term (This Month) - HIGH
1. Add configuration to disable SNMPv2c in production
2. Add SSRF protection for webhook URLs
3. Address Silent Decryption Failures in Credentials module
4. Fix Context Propagation in Storage routines

### Medium Term (Next Quarter)
1. Implement CSP for Web UI (mitigate XSS)
2. Implement refresh token rotation
3. Add encryption key rotation capability
4. Standardize storage layer context usage
5. Add input validation tests for injection attacks

---

*Report reformatted for clarity, grouping completed items and maintaining strict priority order.*
