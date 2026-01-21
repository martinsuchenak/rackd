package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type Handler struct {
	storage storage.ExtendedStorage
}

func NewHandler(s storage.ExtendedStorage) *Handler {
	return &Handler{storage: s}
}

type HandlerOption func(*handlerConfig)

type handlerConfig struct {
	authToken string
}

func WithAuth(token string) HandlerOption {
	return func(c *handlerConfig) {
		c.authToken = token
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, opts ...HandlerOption) {
	cfg := &handlerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	wrap := func(handler http.HandlerFunc) http.HandlerFunc {
		if cfg.authToken != "" {
			return AuthMiddleware(cfg.authToken, handler)
		}
		return handler
	}

	// Datacenter routes
	mux.HandleFunc("GET /api/datacenters", wrap(h.listDatacenters))
	mux.HandleFunc("POST /api/datacenters", wrap(h.createDatacenter))
	mux.HandleFunc("GET /api/datacenters/{id}", wrap(h.getDatacenter))
	mux.HandleFunc("PUT /api/datacenters/{id}", wrap(h.updateDatacenter))
	mux.HandleFunc("DELETE /api/datacenters/{id}", wrap(h.deleteDatacenter))
	mux.HandleFunc("GET /api/datacenters/{id}/devices", wrap(h.getDatacenterDevices))

	// Network routes
	mux.HandleFunc("GET /api/networks", wrap(h.listNetworks))
	mux.HandleFunc("POST /api/networks", wrap(h.createNetwork))
	mux.HandleFunc("GET /api/networks/{id}", wrap(h.getNetwork))
	mux.HandleFunc("PUT /api/networks/{id}", wrap(h.updateNetwork))
	mux.HandleFunc("DELETE /api/networks/{id}", wrap(h.deleteNetwork))
	mux.HandleFunc("GET /api/networks/{id}/devices", wrap(h.getNetworkDevices))
	mux.HandleFunc("GET /api/networks/{id}/utilization", wrap(h.getNetworkUtilization))
	mux.HandleFunc("GET /api/networks/{id}/pools", wrap(h.listNetworkPools))
	mux.HandleFunc("POST /api/networks/{id}/pools", wrap(h.createNetworkPool))

	// Pool routes
	mux.HandleFunc("GET /api/pools/{id}", wrap(h.getNetworkPool))
	mux.HandleFunc("PUT /api/pools/{id}", wrap(h.updateNetworkPool))
	mux.HandleFunc("DELETE /api/pools/{id}", wrap(h.deleteNetworkPool))
	mux.HandleFunc("GET /api/pools/{id}/next-ip", wrap(h.getNextIP))
	mux.HandleFunc("GET /api/pools/{id}/heatmap", wrap(h.getPoolHeatmap))

	// Device routes
	mux.HandleFunc("GET /api/devices", wrap(h.listDevices))
	mux.HandleFunc("POST /api/devices", wrap(h.createDevice))
	mux.HandleFunc("GET /api/devices/{id}", wrap(h.getDevice))
	mux.HandleFunc("PUT /api/devices/{id}", wrap(h.updateDevice))
	mux.HandleFunc("DELETE /api/devices/{id}", wrap(h.deleteDevice))
	mux.HandleFunc("GET /api/devices/search", wrap(h.searchDevices))

	// Relationship routes
	mux.HandleFunc("POST /api/devices/{id}/relationships", wrap(h.addRelationship))
	mux.HandleFunc("GET /api/devices/{id}/relationships", wrap(h.getRelationships))
	mux.HandleFunc("GET /api/devices/{id}/related", wrap(h.getRelatedDevices))
	mux.HandleFunc("DELETE /api/devices/{id}/relationships/{child_id}/{type}", wrap(h.removeRelationship))

	// Discovery routes
	mux.HandleFunc("POST /api/discovery/networks/{id}/scan", wrap(h.startScan))
	mux.HandleFunc("GET /api/discovery/scans", wrap(h.listScans))
	mux.HandleFunc("GET /api/discovery/scans/{id}", wrap(h.getScan))
	mux.HandleFunc("GET /api/discovery/devices", wrap(h.listDiscoveredDevices))
	mux.HandleFunc("POST /api/discovery/devices/{id}/promote", wrap(h.promoteDevice))
	mux.HandleFunc("GET /api/discovery/rules", wrap(h.listDiscoveryRules))
	mux.HandleFunc("POST /api/discovery/rules", wrap(h.createDiscoveryRule))
	mux.HandleFunc("GET /api/discovery/rules/{id}", wrap(h.getDiscoveryRule))
	mux.HandleFunc("PUT /api/discovery/rules/{id}", wrap(h.updateDiscoveryRule))
	mux.HandleFunc("DELETE /api/discovery/rules/{id}", wrap(h.deleteDiscoveryRule))

	// Config route
	mux.HandleFunc("GET /api/config", wrap(h.getConfig))
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}

func (h *Handler) internalError(w http.ResponseWriter, err error) {
	log.Error("Internal server error", "error", err)
	h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal Server Error")
}

func parseArrayParam(r *http.Request, name string) []string {
	values := r.URL.Query()[name]
	if len(values) == 0 {
		return nil
	}
	return values
}

func parseIntParam(r *http.Request, name string, defaultValue int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return result
}

// Stub handlers - will be implemented in separate files
func (h *Handler) listDatacenters(w http.ResponseWriter, r *http.Request)     {}
func (h *Handler) createDatacenter(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) getDatacenter(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) updateDatacenter(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) deleteDatacenter(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) getDatacenterDevices(w http.ResponseWriter, r *http.Request) {}

func (h *Handler) listNetworks(w http.ResponseWriter, r *http.Request)        {}
func (h *Handler) createNetwork(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) getNetwork(w http.ResponseWriter, r *http.Request)          {}
func (h *Handler) updateNetwork(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) deleteNetwork(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) getNetworkDevices(w http.ResponseWriter, r *http.Request)   {}
func (h *Handler) getNetworkUtilization(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) listNetworkPools(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) createNetworkPool(w http.ResponseWriter, r *http.Request)   {}

func (h *Handler) getNetworkPool(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) updateNetworkPool(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) deleteNetworkPool(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) getNextIP(w http.ResponseWriter, r *http.Request)         {}
func (h *Handler) getPoolHeatmap(w http.ResponseWriter, r *http.Request)    {}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request)   {}
func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request)  {}
func (h *Handler) getDevice(w http.ResponseWriter, r *http.Request)     {}
func (h *Handler) updateDevice(w http.ResponseWriter, r *http.Request)  {}
func (h *Handler) deleteDevice(w http.ResponseWriter, r *http.Request)  {}
func (h *Handler) searchDevices(w http.ResponseWriter, r *http.Request) {}

func (h *Handler) addRelationship(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) getRelationships(w http.ResponseWriter, r *http.Request)   {}
func (h *Handler) getRelatedDevices(w http.ResponseWriter, r *http.Request)  {}
func (h *Handler) removeRelationship(w http.ResponseWriter, r *http.Request) {}

func (h *Handler) startScan(w http.ResponseWriter, r *http.Request)              {}
func (h *Handler) listScans(w http.ResponseWriter, r *http.Request)              {}
func (h *Handler) getScan(w http.ResponseWriter, r *http.Request)                {}
func (h *Handler) listDiscoveredDevices(w http.ResponseWriter, r *http.Request)  {}
func (h *Handler) promoteDevice(w http.ResponseWriter, r *http.Request)          {}
func (h *Handler) listDiscoveryRules(w http.ResponseWriter, r *http.Request)     {}
func (h *Handler) createDiscoveryRule(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) getDiscoveryRule(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) updateDiscoveryRule(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) deleteDiscoveryRule(w http.ResponseWriter, r *http.Request)    {}

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {}
