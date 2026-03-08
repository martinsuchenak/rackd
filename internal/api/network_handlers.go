package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listNetworks(w http.ResponseWriter, r *http.Request) {
	filter := &model.NetworkFilter{
		Name:         r.URL.Query().Get("name"),
		DatacenterID: r.URL.Query().Get("datacenter_id"),
		VLANID:       parseIntParam(r, "vlan_id", 0),
	}

	networks, err := h.svc.Networks.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, networks)
}

func (h *Handler) createNetwork(w http.ResponseWriter, r *http.Request) {
	var network model.Network
	if err := json.NewDecoder(r.Body).Decode(&network); err != nil {
		h.invalidJSON(w)
		return
	}

	if err := h.svc.Networks.Create(r.Context(), &network); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, network)
}

func (h *Handler) getNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	network, err := h.svc.Networks.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, network)
}

func (h *Handler) updateNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	network, err := h.svc.Networks.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.invalidJSON(w)
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

	if err := h.svc.Networks.Update(r.Context(), network); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, network)
}

func (h *Handler) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	if err := h.svc.Networks.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getNetworkDevices(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	devices, err := h.svc.Networks.GetDevices(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) getNetworkUtilization(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	utilization, err := h.svc.Networks.GetUtilization(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, utilization)
}

func (h *Handler) listNetworkPools(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	pools, err := h.svc.Pools.ListByNetwork(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, pools)
}

func (h *Handler) createNetworkPool(w http.ResponseWriter, r *http.Request) {
	networkID := r.PathValue("id")

	if networkID == "" {
		h.badRequest(w, "ID is required")
		return
	}
	var pool model.NetworkPool
	if err := json.NewDecoder(r.Body).Decode(&pool); err != nil {
		h.invalidJSON(w)
		return
	}
	pool.NetworkID = networkID

	if err := h.svc.Pools.Create(r.Context(), &pool); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, pool)
}

func (h *Handler) getNetworkPool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	pool, err := h.svc.Pools.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, pool)
}

func (h *Handler) updateNetworkPool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	pool, err := h.svc.Pools.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.invalidJSON(w)
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

	if err := h.svc.Pools.Update(r.Context(), pool); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, pool)
}

func (h *Handler) deleteNetworkPool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	if err := h.svc.Pools.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getNextIP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	ip, err := h.svc.Pools.GetNextIP(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"ip": ip})
}

func (h *Handler) getPoolHeatmap(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	heatmap, err := h.svc.Pools.GetHeatmap(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, heatmap)
}

func (h *Handler) searchNetworks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.badRequest(w, "query parameter 'q' is required")
		return
	}

	networks, err := h.svc.Networks.Search(r.Context(), query)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, networks)
}

func (h *Handler) bulkCreateNetworks(w http.ResponseWriter, r *http.Request) {
	var networks []*model.Network
	if err := json.NewDecoder(r.Body).Decode(&networks); err != nil {
		h.invalidJSON(w)
		return
	}
	if len(networks) > 100 {
		h.badRequest(w, "Maximum 100 items allowed in bulk operations")
		return
	}

	result, err := h.svc.Bulk.CreateNetworks(r.Context(), networks)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) bulkDeleteNetworks(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}
	if len(req.IDs) > 100 {
		h.badRequest(w, "Maximum 100 items allowed in bulk operations")
		return
	}

	result, err := h.svc.Bulk.DeleteNetworks(r.Context(), req.IDs)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}
