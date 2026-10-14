package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/internal/metrics"
)

func TestRequestMetricsUseRoutePattern(t *testing.T) {
	mux := http.NewServeMux()
	HandleAPI(mux, "GET /widgets/{id}", func(w http.ResponseWriter, r *http.Request) {})
	handler := LoggingMiddleware(SecurityHeaders(RecordRoutePattern(mux)))

	for i := range 50 {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", fmt.Sprintf("/api/v1/widgets/w-%d", i), nil))
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(fmt.Sprintf("M%d", i), fmt.Sprintf("/random-%d", i), nil))
	}

	out := metrics.Get().Export()
	for _, want := range []string{`route="GET /api/v1/widgets/{id}"`, `route="unmatched"`} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics missing %s", want)
		}
	}
	for _, leaked := range []string{"w-17", "random-", "M17"} {
		if strings.Contains(out, leaked) {
			t.Errorf("metrics contain raw request data %q", leaked)
		}
	}
}
