package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/martinsuchenak/rackd/pkg/extension"
)

// ExtensionRouter mounts an extension's routes under /api/v1 with core's
// authentication and permission checks. It implements extension.Router.
type ExtensionRouter struct {
	mux     *http.ServeMux
	h       *Handler
	feature string
	errs    []error
}

// ExtensionRouter returns a router for the named extension's routes.
func (h *Handler) ExtensionRouter(mux *http.ServeMux, feature string) *ExtensionRouter {
	return &ExtensionRouter{mux: mux, h: h, feature: feature}
}

// Handle implements extension.Router. Routes are served under /api/v1 only; the
// deprecated unversioned /api aliases are for core's pre-v1 routes.
func (r *ExtensionRouter) Handle(pattern string, handler http.HandlerFunc, opts ...extension.Option) {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok || method == "" || !strings.HasPrefix(path, "/") {
		r.errs = append(r.errs, fmt.Errorf("route %q must be \"METHOD /path\"", pattern))
		return
	}
	o := extension.ApplyOptions(opts)
	if o.Public && o.Permission != nil {
		r.errs = append(r.errs, fmt.Errorf("route %q is Public but requires %s, which needs a caller", pattern, o.Permission))
		return
	}

	final := handler
	if perm := o.Permission; perm != nil {
		next := final
		final = func(w http.ResponseWriter, req *http.Request) {
			if err := r.h.svc.Authorize(req.Context(), perm.Resource, perm.Action); err != nil {
				extension.WriteError(w, err)
				return
			}
			next(w, req)
		}
	}
	if o.Public {
		final = LimitBody(final)
	} else {
		final = r.h.authenticated(final)
	}

	defer func() {
		// ServeMux panics on a pattern that conflicts with one already
		// registered, such as a core route.
		if rec := recover(); rec != nil {
			r.errs = append(r.errs, fmt.Errorf("route %q: %v", pattern, rec))
		}
	}()
	r.mux.HandleFunc(method+" "+APIPrefix+path, final)
}

// Err reports the routes that could not be registered.
func (r *ExtensionRouter) Err() error {
	if len(r.errs) == 0 {
		return nil
	}
	return fmt.Errorf("extension %s: %w", r.feature, errors.Join(r.errs...))
}
