package api

import (
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
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
		h.writeError(w, http.StatusBadRequest, "MISSING_QUERY", "query parameter 'q' is required")
		return
	}

	var results []SearchResult

	if h.svc != nil && h.svc.Devices != nil {
		devices, err := h.svc.Devices.Search(r.Context(), query)
		if err == nil {
			for i := range devices {
				results = append(results, SearchResult{
					Type:   "device",
					Device: &devices[i],
				})
			}
		}
	}

	if h.svc != nil && h.svc.Networks != nil {
		networks, err := h.svc.Networks.Search(r.Context(), query)
		if err == nil {
			for i := range networks {
				results = append(results, SearchResult{
					Type:    "network",
					Network: &networks[i],
				})
			}
		}
	}

	if h.svc != nil && h.svc.Datacenters != nil {
		datacenters, err := h.svc.Datacenters.Search(r.Context(), query)
		if err == nil {
			for i := range datacenters {
				results = append(results, SearchResult{
					Type:       "datacenter",
					Datacenter: &datacenters[i],
				})
			}
		}
	}

	h.writeJSON(w, http.StatusOK, SearchResponse{Results: results})
}
