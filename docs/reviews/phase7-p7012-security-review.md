# Security Review: P7-012 Main Application

**Date:** 2026-01-22
**Reviewer:** AI Assistant
**Files Reviewed:**
- `webui/src/app.ts`
- `webui/src/index.html`

## Summary

The main application entry point implementation is generally secure with good practices. A few low-severity items identified.

## Findings

### LOW-001: Global Window Object Exposure

**Severity:** Low
**Location:** `webui/src/app.ts:79-82`

```typescript
window.rackdConfig = await api.getConfig();
window.Alpine = Alpine;
```

**Issue:** Exposing `rackdConfig` and `Alpine` on the global window object allows browser console access to configuration data and Alpine internals.

**Risk:** An attacker with XSS access could inspect configuration or manipulate Alpine state.

**Mitigation:** This is acceptable for debugging and Enterprise extension support. The config endpoint should not expose sensitive data (verified - only edition, features, nav_items, and user info).

**Status:** Accepted - by design for extensibility.

---

### LOW-002: Theme Preference Stored in localStorage

**Severity:** Low
**Location:** `webui/src/app.ts:27,44`

```typescript
localStorage.getItem('theme')
localStorage.setItem('theme', t);
```

**Issue:** Theme preference stored in localStorage without validation on read.

**Risk:** Minimal - only affects UI theme. Malicious localStorage manipulation would only result in invalid theme defaulting to 'system'.

**Mitigation:** The `getStoredTheme()` function casts to `Theme` type but doesn't validate. However, `applyTheme()` handles any value safely (non-matching values treated as system theme).

**Status:** Acceptable - no security impact.

---

### LOW-003: Dynamic Navigation Items from API

**Severity:** Low
**Location:** `webui/src/index.html:33-38`

```html
<template x-for="item in items" :key="item.id">
  <a :href="item.path" x-text="item.label">
```

**Issue:** Navigation items including `path` come from `/api/config` endpoint.

**Risk:** If the backend is compromised, malicious navigation paths could be injected.

**Mitigation:** 
- Alpine's `x-text` safely escapes content (no XSS via label)
- The `:href` binding could allow `javascript:` URLs if backend is compromised
- Backend should validate nav_items paths are relative URLs

**Recommendation:** Consider adding client-side validation that paths start with `/`.

**Status:** Low risk - requires backend compromise.

---

### INFO-001: No CSP Meta Tag

**Severity:** Informational
**Location:** `webui/src/index.html`

**Issue:** No Content-Security-Policy meta tag in HTML.

**Mitigation:** CSP should be set via HTTP headers by the Go server, not in HTML meta tags. This is the correct approach.

**Status:** Not applicable - CSP handled server-side.

---

### INFO-002: Enterprise SSO Component Reference

**Severity:** Informational
**Location:** `webui/src/index.html:95-108`

```html
<template x-if="hasFeature('sso')">
  <div x-data="sso" class="flex items-center">
```

**Issue:** References `sso` Alpine component that doesn't exist in OSS codebase.

**Risk:** None - Alpine gracefully handles undefined components. The template only renders when `hasFeature('sso')` is true, which requires Enterprise edition.

**Status:** By design - Enterprise extension point.

## Positive Security Practices

1. **XSS Prevention:** Uses Alpine's `x-text` directive which auto-escapes content
2. **CSRF:** API client uses JSON content-type (not form submissions)
3. **Accessibility:** Skip links and ARIA labels don't introduce security issues
4. **No Inline Scripts:** All JavaScript loaded via module, compatible with strict CSP
5. **Search Input:** Uses `type="search"` with proper escaping via `encodeURIComponent` in API client

## Recommendations

1. **Optional:** Add client-side validation for nav item paths:
   ```typescript
   // In nav.ts init()
   const dynamicItems = (this.config?.nav_items ?? [])
     .filter(item => item.path.startsWith('/'));
   ```

2. **Documentation:** Document that `/api/config` should never expose sensitive data.

## Conclusion

**Overall Assessment:** PASS

The implementation follows security best practices for a frontend application. No high or medium severity issues found. The identified low-severity items are acceptable given the architecture and threat model.
