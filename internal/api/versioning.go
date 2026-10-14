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
