package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/metrics"
)

func TestBuildHandlerLabelsMetricsByRoute(t *testing.T) {
	mux := http.NewServeMux()
	api.HandleAPI(mux, "GET /gadgets/{id}", func(http.ResponseWriter, *http.Request) {})
	cfg := &config.Config{RateLimitEnabled: true, RateLimitRequests: 2, RateLimitWindow: time.Minute}
	handler := buildHandler(mux, cfg, api.NewTrustedProxyResolver(false, ""))

	statuses := map[int]int{}
	for _, path := range []string{"/api/v1/gadgets/g-1", "/api/v1/gadgets/g-2", "/api/v1/gadgets/g-3"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		statuses[rec.Code]++
	}
	if statuses[http.StatusOK] != 2 || statuses[http.StatusTooManyRequests] != 1 {
		t.Fatalf("statuses = %v, want two 200s and one 429", statuses)
	}

	out := metrics.Get().Export()
	for _, want := range []string{`route="GET /api/v1/gadgets/{id}"`, `route="unrouted"`} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics missing %s", want)
		}
	}
	if strings.Contains(out, "g-1") {
		t.Error("metrics contain a raw path")
	}
}
