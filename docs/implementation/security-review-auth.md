# Security Review: Authentication & User Module

**Date:** 2026-02-06
**Scope:** Login, logout, session management, user CRUD, API key authentication, password handling, middleware, frontend auth flow

## Architecture Overview

The system uses httpOnly cookie-based session authentication with in-memory sessions. Passwords are hashed with bcrypt (cost 12). Session tokens are 32-byte `crypto/rand` values. API key authentication uses `Authorization: Bearer <key>` headers with constant-time comparison.

---

## CRITICAL

### 1. API Key Lookup Is Not Constant-Time (SQL `WHERE key = ?`)

- **Status:** [x] Fixed
- **File:** `internal/storage/apikey_sqlite.go:87-93`
- **Description:** `GetAPIKeyByKey` does a direct SQL string comparison (`WHERE key = ?`). This is a timing side-channel -- the database comparison time may vary based on how many characters match. The in-memory `Authenticator` in `internal/auth/apikey.go:46` correctly uses `subtle.ConstantTimeCompare`, but the middleware at `internal/api/middleware.go:82` calls the SQLite store instead, bypassing the constant-time check entirely.
- **Impact:** An attacker could potentially derive a valid API key byte-by-byte via timing analysis.
- **Fix:** Added `crypto/subtle.ConstantTimeCompare` verification in both `AuthMiddleware` and `AuthMiddlewareWithSessions` after DB lookup.
- **Fix effort:** Medium

### 2. No Authorization / RBAC -- Any Authenticated User Can Do Everything

- **Status:** [ ] Open
- **File:** `internal/api/user_handlers.go`, `internal/api/handlers.go:222-228`
- **Description:** All user management routes (create, update, delete users, change any user's password) only require authentication (`wrapAuth`), not admin privileges. A regular non-admin user can: create new admin users, delete other users, change anyone's password, list all users. The `IsAdmin` field exists on sessions but is never checked in any handler.
- **Impact:** Complete privilege escalation. Any authenticated user has full admin capabilities.
- **Fix effort:** Medium

### 3. No Rate Limiting on Login Endpoint

- **Status:** [x] Fixed
- **File:** `internal/api/handlers.go:218`, `internal/api/ratelimit.go:132`
- **Description:** The login endpoint (`POST /api/auth/login`) uses only `LimitBody` -- it does not go through `wrap()` or `wrapAuth()`, so it inherits the global rate limiter from the middleware chain. However, the rate limiter bypasses localhost entirely (`ratelimit.go:133`), and uses the same generous limits as all other endpoints. There is no login-specific rate limiting or account lockout after failed attempts.
- **Impact:** Brute-force attacks on passwords are feasible, especially from localhost or when behind a reverse proxy.
- **Fix:** Added dedicated `LoginRateLimitMiddleware` with strict per-IP limits (default: 5 requests/minute). Does NOT bypass localhost. Configurable via `LOGIN_RATE_LIMIT_REQUESTS` and `LOGIN_RATE_LIMIT_WINDOW` env vars.
- **Fix effort:** Low

### 4. Token Stored in localStorage -- Vulnerable to XSS

- **Status:** [x] Fixed
- **File:** `webui/src/components/login.ts:69`
- **Description:** Session tokens are stored in `localStorage`, which is accessible to any JavaScript running on the page. Combined with the CSP allowing `'unsafe-eval'` and `'unsafe-inline'` (`middleware.go:178`), an XSS vulnerability would immediately compromise all session tokens.
- **Impact:** Token theft via XSS. `httpOnly` cookies would be significantly more secure.
- **Fix:** Moved session tokens to `httpOnly`, `Secure`, `SameSite=Lax` cookies. Token no longer returned in login response body. Frontend no longer stores or manages tokens. Cookie security configurable via `COOKIE_SECURE` env var. Also fixed logout to call server endpoint for proper session invalidation.
- **Fix effort:** Medium

---

## HIGH

### 5. CSP Allows `unsafe-eval` and `unsafe-inline`

- **Status:** [ ] Open
- **File:** `internal/api/middleware.go:178`
- **Description:** The Content-Security-Policy header includes both `'unsafe-eval'` and `'unsafe-inline'` for scripts. This substantially weakens XSS protection. The comment acknowledges this is for Alpine.js -- the Alpine CSP build (`@alpinejs/csp`) would eliminate this need.
- **Fix effort:** Medium

### 6. User Enumeration via Different Error Responses

- **Status:** [x] Fixed
- **File:** `internal/api/auth_handlers.go:38-42`
- **Description:** When a user exists but is inactive, the login endpoint returns `403 USER_INACTIVE` instead of the generic `401 INVALID_CREDENTIALS`. This reveals that the username exists and the account is deactivated.
- **Impact:** Attackers can enumerate valid usernames and identify deactivated accounts.
- **Fix:** Inactive users now return the same `401 INVALID_CREDENTIALS` response as invalid username/password.
- **Fix effort:** Low

### 7. Session Refresh Race Condition

- **Status:** [x] Fixed
- **File:** `internal/api/middleware.go:150-152`
- **Description:** Session refresh is done asynchronously in a goroutine. If the session expires between `GetSession()` and `RefreshSession()`, the refresh silently fails. More importantly, there is a TOCTOU race -- the session could be invalidated (e.g., by password change) between the get and the refresh, allowing continued access briefly.
- **Impact:** Brief continued access after session invalidation.
- **Fix:** Session refresh is now called synchronously in the middleware (as part of the cookie-based session migration). The goroutine was removed.
- **Fix effort:** Low

### 8. X-Forwarded-For Header Spoofing for Rate Limiting

- **Status:** [x] Fixed
- **File:** `internal/api/ratelimit.go:170-174`
- **Description:** `getClientIP` trusts `X-Forwarded-For` and `X-Real-IP` headers directly. An attacker can set these headers to arbitrary values to bypass per-IP rate limiting. This should only be trusted when the app is behind a known reverse proxy.
- **Impact:** Rate limit bypass by spoofing proxy headers.
- **Fix:** Added `TRUST_PROXY` config option (default: `false`). `getClientIP` now only reads `X-Forwarded-For` and `X-Real-IP` headers when `trustProxy` is explicitly enabled. Both `RateLimitMiddleware` and `LoginRateLimitMiddleware` respect this setting.
- **Fix effort:** Low

### 9. Metrics Endpoint Exposed Without Auth

- **Status:** [x] Fixed
- **File:** `internal/api/handlers.go:232-235`
- **Description:** `/metrics` was exposed without authentication, potentially leaking operational details (request counts, latencies, error rates) that aid reconnaissance. `/healthz` and `/readyz` remain unauthenticated as they are standard health check endpoints used by load balancers and orchestrators.
- **Impact:** Information disclosure of operational metrics.
- **Fix:** Changed `/metrics` route to use `wrap(h.metricsHandler)` which applies authentication middleware.
- **Fix effort:** Low

### 10. In-Memory Sessions Lost on Restart

- **Status:** [ ] Open
- **File:** `internal/auth/session.go`
- **Description:** All sessions are stored in a `map[string]*Session`. A server restart logs out all users. In a multi-instance deployment, sessions are not shared between instances, meaning a user authenticated on instance A will be rejected by instance B.
- **Fix effort:** High

---

## MEDIUM

### 11. No Maximum Password Length Check

- **Status:** [ ] Open
- **File:** `internal/api/auth_handlers.go`, `internal/auth/password.go:18`
- **Description:** bcrypt has a 72-byte input limit. Passwords longer than 72 bytes are silently truncated. There is no server-side max length validation. An attacker could also submit extremely long passwords to cause CPU-intensive hashing (though `LimitBody` mitigates this somewhat at 1MB).
- **Fix effort:** Low

### 12. Password Complexity -- Length Only

- **Status:** [ ] Open
- **File:** `internal/api/user_handlers.go:61`
- **Description:** Only a minimum of 8 characters is enforced. No requirement for mixed case, digits, or special characters. While length is the most important factor, a password like "aaaaaaaa" would be accepted.
- **Fix effort:** Low

### 13. API Keys Returned in Full via List Endpoint

- **Status:** [ ] Open
- **File:** `internal/storage/apikey_sqlite.go:117-118`
- **Description:** `ListAPIKeys` returns the full key value in the response. Once listed, any authenticated user (see issue #2) can see all API keys in plaintext. Keys should be masked/truncated in list responses.
- **Fix effort:** Low

### 14. No Audit Logging for Failed Login Attempts

- **Status:** [ ] Open
- **File:** `internal/api/auth_handlers.go:33-46`
- **Description:** Failed logins are logged via `log.Warn` but not recorded in the audit trail. Security monitoring tools typically need structured audit records of authentication failures.
- **Fix effort:** Low

### 15. Self-Deletion Check May Be Bypassed

- **Status:** [ ] Open
- **File:** `internal/api/user_handlers.go:164`
- **Description:** The self-deletion check uses `contextKey(SessionContextKey)` -- note the type cast. But the middleware stores the session using `SessionContextKey` directly (type `contextKey`). Since `contextKey` is already a `contextKey`, wrapping it again creates a different key. This needs verification -- if the types don't match, the self-deletion guard is bypassed.
- **Fix effort:** Low

### 16. HSTS Missing Behind Reverse Proxy

- **Status:** [ ] Open
- **File:** `internal/api/middleware.go:181-183`
- **Description:** HSTS is only set when `r.TLS != nil`. If the app runs behind a TLS-terminating proxy (common), `r.TLS` will be nil and HSTS won't be set. This should be configurable.
- **Fix effort:** Low

---

## LOW

### 17. Token Expiry Exposed in Response

- **Status:** [ ] Open
- **File:** `internal/api/auth_handlers.go:67`
- **Description:** Exposing `ExpiresAt` tells attackers exactly when sessions expire, allowing them to time session reuse attacks.
- **Fix effort:** Low

### 18. No `Cache-Control` Header on Auth Responses

- **Status:** [ ] Open
- **Description:** API responses containing tokens or user data don't set `Cache-Control: no-store`, which could result in sensitive data being cached by browsers or proxies.
- **Fix effort:** Low

### 19. Username Not Sanitized/Validated

- **Status:** [ ] Open
- **File:** `internal/api/auth_handlers.go:21-24`, `internal/api/user_handlers.go:56-58`
- **Description:** Usernames only check for emptiness. No validation for length, allowed characters, or format. Special characters in usernames could cause issues in logging or downstream systems.
- **Fix effort:** Low

---

## Summary

| # | Severity | Issue | Effort | Status |
|---|----------|-------|--------|--------|
| 1 | CRITICAL | API key timing side-channel in SQL lookup | Medium | **Fixed** |
| 2 | CRITICAL | No RBAC -- any user can manage all users | Medium | Open |
| 3 | CRITICAL | No login-specific rate limiting / lockout | Low | **Fixed** |
| 4 | CRITICAL | Token in localStorage + weak CSP | Medium | **Fixed** |
| 5 | HIGH | CSP `unsafe-eval` / `unsafe-inline` | Medium | Open |
| 6 | HIGH | User enumeration via inactive account error | Low | **Fixed** |
| 7 | HIGH | Session refresh race condition | Low | **Fixed** |
| 8 | HIGH | X-Forwarded-For spoofing bypasses rate limit | Low | **Fixed** |
| 9 | HIGH | Metrics endpoint unauthenticated | Low | **Fixed** |
| 10 | HIGH | In-memory sessions lost on restart | High | Open |
| 11 | MEDIUM | No bcrypt 72-byte max password check | Low | Open |
| 12 | MEDIUM | Weak password policy | Low | Open |
| 13 | MEDIUM | API keys exposed in full via list | Low | Open |
| 14 | MEDIUM | No audit trail for failed logins | Low | Open |
| 15 | MEDIUM | Self-deletion check may be bypassed | Low | Open |
| 16 | MEDIUM | HSTS missing behind reverse proxy | Low | Open |
| 17 | LOW | Token expiry exposed in response | Low | Open |
| 18 | LOW | No Cache-Control on auth responses | Low | Open |
| 19 | LOW | No username format validation | Low | Open |
