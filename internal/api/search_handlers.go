package api

import (
	"errors"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type SearchResult struct {
	Type       string            `json:"type"`
	Device     *model.Device     `json:"device,omitempty"`
	Network    *model.Network    `json:"network,omitempty"`
	Datacenter *model.Datacenter `json:"datacenter,omitempty"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.badRequest(w, "query parameter 'q' is required")
		return
	}

	if err := h.svc.Authorize(r.Context(), "search", "read"); err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Each kind needs its own list permission; kinds the caller can't list are
	// left out. Any other failure is an error, not an empty result.
	var results []SearchResult
	skip := func(err error) bool { return errors.Is(err, service.ErrForbidden) }

	devices, err := h.svc.Devices.Search(r.Context(), query)
	if err != nil && !skip(err) {
		h.handleServiceError(w, err)
		return
	}
	for i := range devices {
		results = append(results, SearchResult{Type: "device", Device: &devices[i]})
	}

	networks, err := h.svc.Networks.Search(r.Context(), query)
	if err != nil && !skip(err) {
		h.handleServiceError(w, err)
		return
	}
	for i := range networks {
		results = append(results, SearchResult{Type: "network", Network: &networks[i]})
	}

	datacenters, err := h.svc.Datacenters.Search(r.Context(), query)
	if err != nil && !skip(err) {
		h.handleServiceError(w, err)
		return
	}
	for i := range datacenters {
		results = append(results, SearchResult{Type: "datacenter", Datacenter: &datacenters[i]})
	}

	h.writeJSON(w, http.StatusOK, SearchResponse{Results: results})
}
