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

	if h.svc != nil && h.svc.Networks != nil {
		networks, err := h.svc.Networks.List(r.Context(), filter)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, networks)
		return
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

	if h.svc != nil && h.svc.Networks != nil {
		if err := h.svc.Networks.Create(r.Context(), &network); err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusCreated, network)
		return
	}

	if errs := ValidateNetwork(&network); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}
	if err := h.store.CreateNetwork(h.auditContext(r), &network); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, network)
}

func (h *Handler) getNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Networks != nil {
		network, err := h.svc.Networks.Get(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, network)
		return
	}

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

	// Fetch through service so RBAC is enforced on the read too
	var network *model.Network
	var err error
	if h.svc != nil && h.svc.Networks != nil {
		network, err = h.svc.Networks.Get(r.Context(), id)
	} else {
		network, err = h.store.GetNetwork(id)
	}
	if err != nil {
		if h.svc != nil && h.svc.Networks != nil {
			h.handleServiceError(w, err)
		} else if errors.Is(err, storage.ErrNetworkNotFound) {
			h.writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found")
		} else {
			h.internalError(w, err)
		}
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

	if h.svc != nil && h.svc.Networks != nil {
		if err := h.svc.Networks.Update(r.Context(), network); err != nil {
			h.handleServiceError(w, err)
			return
		}
	} else {
		if errs := ValidateNetwork(network); len(errs) > 0 {
			h.writeValidationErrors(w, errs)
			return
		}
		if err := h.store.UpdateNetwork(h.auditContext(r), network); err != nil {
			h.internalError(w, err)
			return
		}
	}
	h.writeJSON(w, http.StatusOK, network)
}

func (h *Handler) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Networks != nil {
		if err := h.svc.Networks.Delete(r.Context(), id); err != nil {
			h.handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.store.DeleteNetwork(h.auditContext(r), id); err != nil {
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

	if h.svc != nil && h.svc.Networks != nil {
		devices, err := h.svc.Networks.GetDevices(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, devices)
		return
	}

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

	if h.svc != nil && h.svc.Networks != nil {
		utilization, err := h.svc.Networks.GetUtilization(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, utilization)
		return
	}

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

	if h.svc != nil && h.svc.Pools != nil {
		// Verify network exists via service layer
		if _, err := h.svc.Networks.Get(r.Context(), id); err != nil {
			h.handleServiceError(w, err)
			return
		}
		filter := &model.NetworkPoolFilter{NetworkID: id}
		pools, err := h.svc.Pools.List(r.Context(), filter)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, pools)
		return
	}

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

	if h.svc != nil && h.svc.Pools != nil {
		// Verify network exists via service layer
		if _, err := h.svc.Networks.Get(r.Context(), networkID); err != nil {
			h.handleServiceError(w, err)
			return
		}

		var pool model.NetworkPool
		if err := json.NewDecoder(r.Body).Decode(&pool); err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
			return
		}
		pool.NetworkID = networkID

		if err := h.svc.Pools.Create(r.Context(), &pool); err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusCreated, pool)
		return
	}

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

	if err := h.store.CreateNetworkPool(h.auditContext(r), &pool); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, pool)
}

func (h *Handler) getNetworkPool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Pools != nil {
		pool, err := h.svc.Pools.Get(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, pool)
		return
	}

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

	// Fetch through service so RBAC is enforced on the read too
	var pool *model.NetworkPool
	var err error
	if h.svc != nil && h.svc.Pools != nil {
		pool, err = h.svc.Pools.Get(r.Context(), id)
	} else {
		pool, err = h.store.GetNetworkPool(id)
	}
	if err != nil {
		if h.svc != nil && h.svc.Pools != nil {
			h.handleServiceError(w, err)
		} else if errors.Is(err, storage.ErrPoolNotFound) {
			h.writeError(w, http.StatusNotFound, "POOL_NOT_FOUND", "Pool not found")
		} else {
			h.internalError(w, err)
		}
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

	if h.svc != nil && h.svc.Pools != nil {
		if err := h.svc.Pools.Update(r.Context(), pool); err != nil {
			h.handleServiceError(w, err)
			return
		}
	} else {
		if errs := ValidateNetworkPool(pool); len(errs) > 0 {
			h.writeValidationErrors(w, errs)
			return
		}
		if err := h.store.UpdateNetworkPool(h.auditContext(r), pool); err != nil {
			h.internalError(w, err)
			return
		}
	}
	h.writeJSON(w, http.StatusOK, pool)
}

func (h *Handler) deleteNetworkPool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Pools != nil {
		if err := h.svc.Pools.Delete(r.Context(), id); err != nil {
			h.handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.store.DeleteNetworkPool(h.auditContext(r), id); err != nil {
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

	if h.svc != nil && h.svc.Pools != nil {
		ip, err := h.svc.Pools.GetNextIP(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]string{"ip": ip})
		return
	}

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

	if h.svc != nil && h.svc.Pools != nil {
		heatmap, err := h.svc.Pools.GetHeatmap(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, heatmap)
		return
	}

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

	if h.svc != nil && h.svc.Networks != nil {
		networks, err := h.svc.Networks.Search(r.Context(), query)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, networks)
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

	result, err := h.store.BulkCreateNetworks(h.auditContext(r), networks)
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

	result, err := h.store.BulkDeleteNetworks(h.auditContext(r), req.IDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
