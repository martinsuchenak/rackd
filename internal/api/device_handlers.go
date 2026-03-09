package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	filter := &model.DeviceFilter{
		Pagination:   parsePagination(r),
		Tags:         parseArrayParam(r, "tags"),
		DatacenterID: r.URL.Query().Get("datacenter_id"),
		NetworkID:    r.URL.Query().Get("network_id"),
		PoolID:       r.URL.Query().Get("pool_id"),
		Status:       model.DeviceStatus(r.URL.Query().Get("status")),
	}
	// Handle stale filter - if stale=true, use default of 7 days
	if r.URL.Query().Get("stale") == "true" {
		filter.StaleDays = parseIntParam(r, "stale_days", 7)
	} else if staleDays := parseIntParam(r, "stale_days", 0); staleDays > 0 {
		filter.StaleDays = staleDays
	}
	devices, err := h.svc.Devices.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request) {
	var device model.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		h.invalidJSON(w)
		return
	}
	if errs := ValidateDevice(&device); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}

	if err := h.svc.Devices.Create(r.Context(), &device); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, device)
}

func (h *Handler) getDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}

	device, err := h.svc.Devices.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, device)
}

func (h *Handler) updateDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}

	device, err := h.svc.Devices.Get(r.Context(), id)
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
		device.Name = name
	}
	if hostname, ok := updates["hostname"].(string); ok {
		device.Hostname = hostname
	}
	if description, ok := updates["description"].(string); ok {
		device.Description = description
	}
	if makeModel, ok := updates["make_model"].(string); ok {
		device.MakeModel = makeModel
	}
	if os, ok := updates["os"].(string); ok {
		device.OS = os
	}
	if datacenterID, ok := updates["datacenter_id"].(string); ok {
		device.DatacenterID = datacenterID
	}
	if username, ok := updates["username"].(string); ok {
		device.Username = username
	}
	if location, ok := updates["location"].(string); ok {
		device.Location = location
	}
	if status, ok := updates["status"].(string); ok {
		device.Status = model.DeviceStatus(status)
	}
	if decommissionDate, ok := updates["decommission_date"].(string); ok && decommissionDate != "" {
		t, err := time.Parse(time.RFC3339, decommissionDate)
		if err == nil {
			device.DecommissionDate = &t
		}
	}
	if tags, ok := updates["tags"].([]any); ok {
		device.Tags = toStringSlice(tags)
	}
	if domains, ok := updates["domains"].([]any); ok {
		device.Domains = toStringSlice(domains)
	}
	if addresses, ok := updates["addresses"].([]any); ok {
		device.Addresses = toAddressSlice(addresses)
	}
	if customFields, ok := updates["custom_fields"].([]any); ok {
		device.CustomFields = toCustomFieldSlice(customFields)
	}

	if errs := ValidateDevice(device); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}

	if err := h.svc.Devices.Update(r.Context(), device); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, device)
}

func (h *Handler) deleteDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}

	if err := h.svc.Devices.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) searchDevices(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.badRequest(w, "Query parameter 'q' is required")
		return
	}
	if len(query) > 256 {
		h.badRequest(w, "Query parameter must be 256 characters or less")
		return
	}

	devices, err := h.svc.Devices.Search(r.Context(), query)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}

func toStringSlice(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func toAddressSlice(arr []any) []model.Address {
	result := make([]model.Address, 0, len(arr))
	for _, v := range arr {
		if m, ok := v.(map[string]any); ok {
			addr := model.Address{}
			if ip, ok := m["ip"].(string); ok {
				addr.IP = ip
			}
			if port, ok := m["port"].(float64); ok && port > 0 {
				p := int(port)
				addr.Port = &p
			}
			if t, ok := m["type"].(string); ok {
				addr.Type = t
			}
			if label, ok := m["label"].(string); ok {
				addr.Label = label
			}
			if networkID, ok := m["network_id"].(string); ok {
				addr.NetworkID = networkID
			}
			if switchPort, ok := m["switch_port"].(string); ok {
				addr.SwitchPort = switchPort
			}
			if poolID, ok := m["pool_id"].(string); ok {
				addr.PoolID = poolID
			}
			result = append(result, addr)
		}
	}
	return result
}

func toCustomFieldSlice(arr []any) []model.CustomFieldValueInput {
	result := make([]model.CustomFieldValueInput, 0, len(arr))
	for _, v := range arr {
		if m, ok := v.(map[string]any); ok {
			cf := model.CustomFieldValueInput{}
			if fieldID, ok := m["field_id"].(string); ok {
				cf.FieldID = fieldID
			}
			if value, ok := m["value"]; ok {
				cf.Value = value
			}
			result = append(result, cf)
		}
	}
	return result
}

func (h *Handler) bulkCreateDevices(w http.ResponseWriter, r *http.Request) {
	var devices []*model.Device
	if err := json.NewDecoder(r.Body).Decode(&devices); err != nil {
		h.invalidJSON(w)
		return
	}
	if len(devices) > 100 {
		h.badRequest(w, "Maximum 100 items allowed in bulk operations")
		return
	}

	result, err := h.svc.Bulk.CreateDevices(r.Context(), devices)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) bulkUpdateDevices(w http.ResponseWriter, r *http.Request) {
	var devices []*model.Device
	if err := json.NewDecoder(r.Body).Decode(&devices); err != nil {
		h.invalidJSON(w)
		return
	}
	if len(devices) > 100 {
		h.badRequest(w, "Maximum 100 items allowed in bulk operations")
		return
	}

	result, err := h.svc.Bulk.UpdateDevices(r.Context(), devices)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) bulkDeleteDevices(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.svc.Bulk.DeleteDevices(r.Context(), req.IDs)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) bulkAddTags(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
		Tags      []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}
	if len(req.DeviceIDs) > 100 {
		h.badRequest(w, "Maximum 100 items allowed in bulk operations")
		return
	}

	result, err := h.svc.Bulk.AddTags(r.Context(), req.DeviceIDs, req.Tags)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) bulkRemoveTags(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
		Tags      []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}
	if len(req.DeviceIDs) > 100 {
		h.badRequest(w, "Maximum 100 items allowed in bulk operations")
		return
	}

	result, err := h.svc.Bulk.RemoveTags(r.Context(), req.DeviceIDs, req.Tags)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getDeviceStatusCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.svc.Devices.GetStatusCounts(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Ensure all status keys are present with 0 if no devices
	result := map[string]int{
		"planned":        0,
		"active":         0,
		"maintenance":    0,
		"decommissioned": 0,
	}
	for status, count := range counts {
		result[string(status)] = count
	}

	h.writeJSON(w, http.StatusOK, result)
}
