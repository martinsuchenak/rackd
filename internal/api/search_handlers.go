package api

import (
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

type SearchResult struct {
	Type       string              `json:"type"`
	Device     *model.Device       `json:"device,omitempty"`
	Network    *model.Network      `json:"network,omitempty"`
	Datacenter *model.Datacenter   `json:"datacenter,omitempty"`
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

	// Search devices
	devices, err := h.storage.SearchDevices(query)
	if err == nil {
		for i := range devices {
			results = append(results, SearchResult{
				Type:   "device",
				Device: &devices[i],
			})
		}
	}

	// Search networks
	networks, err := h.storage.SearchNetworks(query)
	if err == nil {
		for i := range networks {
			results = append(results, SearchResult{
				Type:    "network",
				Network: &networks[i],
			})
		}
	}

	// Search datacenters
	datacenters, err := h.storage.SearchDatacenters(query)
	if err == nil {
		for i := range datacenters {
			results = append(results, SearchResult{
				Type:       "datacenter",
				Datacenter: &datacenters[i],
			})
		}
	}

	h.writeJSON(w, http.StatusOK, SearchResponse{Results: results})
}
