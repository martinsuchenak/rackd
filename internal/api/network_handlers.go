package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func (h *Handler) listNetworks(w http.ResponseWriter, r *http.Request) {
	filter := &model.NetworkFilter{
		Name:         r.URL.Query().Get("name"),
		DatacenterID: r.URL.Query().Get("datacenter_id"),
		VLANID:       parseIntParam(r, "vlan_id", 0),
	}
	networks, err := h.store.ListNetworks(filter)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, networks)
}

func (h *Handler) createNetwork(w http.ResponseWriter, r *http.Request) {
	var network model.Network
	if err := json.NewDecoder(r.Body).Decode(&network); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}
	if errs := ValidateNetwork(&network); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}
	if err := h.store.CreateNetwork(&network); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, network)
}

func (h *Handler) getNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network, err := h.store.GetNetwork(id)
	if err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			h.writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, network)
}

func (h *Handler) updateNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network, err := h.store.GetNetwork(id)
	if err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			h.writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found")
			return
		}
		h.internalError(w, err)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	if name, ok := updates["name"].(string); ok {
		network.Name = name
	}
	if subnet, ok := updates["subnet"].(string); ok {
		network.Subnet = subnet
	}
	if vlanID, ok := updates["vlan_id"].(float64); ok {
		network.VLANID = int(vlanID)
	}
	if datacenterID, ok := updates["datacenter_id"].(string); ok {
		network.DatacenterID = datacenterID
	}
	if description, ok := updates["description"].(string); ok {
		network.Description = description
	}

	if errs := ValidateNetwork(network); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}

	if err := h.store.UpdateNetwork(network); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, network)
}

func (h *Handler) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteNetwork(id); err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			h.writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getNetworkDevices(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.store.GetNetwork(id); err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			h.writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found")
			return
		}
		h.internalError(w, err)
		return
	}
	devices, err := h.store.GetNetworkDevices(id)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) getNetworkUtilization(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	utilization, err := h.store.GetNetworkUtilization(id)
	if err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			h.writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, utilization)
}

func (h *Handler) listNetworkPools(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.store.GetNetwork(id); err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			h.writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found")
			return
		}
		h.internalError(w, err)
		return
	}
	filter := &model.NetworkPoolFilter{NetworkID: id}
	pools, err := h.store.ListNetworkPools(filter)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, pools)
}

func (h *Handler) createNetworkPool(w http.ResponseWriter, r *http.Request) {
	networkID := r.PathValue("id")
	if _, err := h.store.GetNetwork(networkID); err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			h.writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found")
			return
		}
		h.internalError(w, err)
		return
	}

	var pool model.NetworkPool
	if err := json.NewDecoder(r.Body).Decode(&pool); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}
	pool.NetworkID = networkID

	if errs := ValidateNetworkPool(&pool); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}

	if err := h.store.CreateNetworkPool(&pool); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, pool)
}

func (h *Handler) getNetworkPool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	pool, err := h.store.GetNetworkPool(id)
	if err != nil {
		if errors.Is(err, storage.ErrPoolNotFound) {
			h.writeError(w, http.StatusNotFound, "POOL_NOT_FOUND", "Pool not found")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, pool)
}

func (h *Handler) updateNetworkPool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	pool, err := h.store.GetNetworkPool(id)
	if err != nil {
		if errors.Is(err, storage.ErrPoolNotFound) {
			h.writeError(w, http.StatusNotFound, "POOL_NOT_FOUND", "Pool not found")
			return
		}
		h.internalError(w, err)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	if name, ok := updates["name"].(string); ok {
		pool.Name = name
	}
	if startIP, ok := updates["start_ip"].(string); ok {
		pool.StartIP = startIP
	}
	if endIP, ok := updates["end_ip"].(string); ok {
		pool.EndIP = endIP
	}
	if description, ok := updates["description"].(string); ok {
		pool.Description = description
	}
	if tags, ok := updates["tags"].([]any); ok {
		pool.Tags = make([]string, len(tags))
		for i, t := range tags {
			if s, ok := t.(string); ok {
				pool.Tags[i] = s
			}
		}
	}

	if errs := ValidateNetworkPool(pool); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}

	if err := h.store.UpdateNetworkPool(pool); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, pool)
}

func (h *Handler) deleteNetworkPool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteNetworkPool(id); err != nil {
		if errors.Is(err, storage.ErrPoolNotFound) {
			h.writeError(w, http.StatusNotFound, "POOL_NOT_FOUND", "Pool not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getNextIP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ip, err := h.store.GetNextAvailableIP(id)
	if err != nil {
		if errors.Is(err, storage.ErrPoolNotFound) {
			h.writeError(w, http.StatusNotFound, "POOL_NOT_FOUND", "Pool not found")
			return
		}
		if errors.Is(err, storage.ErrIPNotAvailable) {
			h.writeError(w, http.StatusConflict, "IP_NOT_AVAILABLE", "No IP addresses available")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"ip": ip})
}

func (h *Handler) getPoolHeatmap(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	heatmap, err := h.store.GetPoolHeatmap(id)
	if err != nil {
		if errors.Is(err, storage.ErrPoolNotFound) {
			h.writeError(w, http.StatusNotFound, "POOL_NOT_FOUND", "Pool not found")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, heatmap)
}

func (h *Handler) searchNetworks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_QUERY", "query parameter 'q' is required")
		return
	}

	networks, err := h.store.SearchNetworks(query)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, networks)
}


// bulkCreateNetworks handles POST /api/networks/bulk
func (h *Handler) bulkCreateNetworks(w http.ResponseWriter, r *http.Request) {
	var networks []*model.Network
	if err := json.NewDecoder(r.Body).Decode(&networks); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := h.store.BulkCreateNetworks(networks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// bulkDeleteNetworks handles DELETE /api/networks/bulk
func (h *Handler) bulkDeleteNetworks(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := h.store.BulkDeleteNetworks(req.IDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
