package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// APIPrefix is the path prefix of the current, stable API version.
const APIPrefix = "/api/v1"

// legacyPrefix is the unversioned prefix used before v1. Routes under it are
// deprecated aliases of the v1 routes and are removed after legacySunset.
const legacyPrefix = "/api"

var (
	legacyDeprecated = time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	legacySunset     = time.Date(2027, 4, 14, 0, 0, 0, 0, time.UTC)
)

// HandleAPI registers handler for pattern ("METHOD /path") under /api/v1. Until
// legacySunset it is also served under the unversioned /api prefix, with
// Deprecation (RFC 9745), Sunset (RFC 8594) and successor-version Link headers.
func HandleAPI(mux *http.ServeMux, pattern string, handler http.HandlerFunc) {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok || !strings.HasPrefix(path, "/") {
		panic(fmt.Sprintf("api: route pattern %q must be \"METHOD /path\"", pattern))
	}
	mux.HandleFunc(method+" "+APIPrefix+path, handler)
	mux.HandleFunc(method+" "+legacyPrefix+path, deprecatedAlias(handler))
}

func deprecatedAlias(next http.HandlerFunc) http.HandlerFunc {
	deprecation := fmt.Sprintf("@%d", legacyDeprecated.Unix())
	sunset := legacySunset.Format(http.TimeFormat)
	return func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Deprecation", deprecation)
		h.Set("Sunset", sunset)
		successor := APIPrefix + strings.TrimPrefix(r.URL.EscapedPath(), legacyPrefix)
		h.Add("Link", fmt.Sprintf("<%s>; rel=\"successor-version\"", successor))
		next(w, r)
	}
}

// apiFallbackMethods get a catch-all pattern under /api/. A single method-less
// "/api/" pattern would conflict with the web UI's "GET /".
var apiFallbackMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

// registerAPIFallback answers requests under /api/ that match no route with a
// JSON 405 when the path exists for other methods, and a JSON 404 otherwise,
// instead of letting them fall through to the web UI.
func (h *Handler) registerAPIFallback(mux *http.ServeMux) {
	isFallback := func(pattern string) bool { return strings.HasSuffix(pattern, " /api/") }
	fallback := func(w http.ResponseWriter, r *http.Request) {
		var allowed []string
		for _, method := range apiFallbackMethods {
			probe := r.WithContext(r.Context())
			probe.Method = method
			if _, pattern := mux.Handler(probe); pattern != "" && !isFallback(pattern) {
				allowed = append(allowed, method)
				if method == http.MethodGet {
					allowed = append(allowed, http.MethodHead)
				}
			}
		}
		if len(allowed) == 0 {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "no such API endpoint")
			return
		}
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		h.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed for this API endpoint")
	}
	for _, method := range apiFallbackMethods {
		mux.HandleFunc(method+" /api/", fallback)
	}
}
