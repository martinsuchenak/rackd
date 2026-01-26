package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	filter := &model.DeviceFilter{
		Tags:         parseArrayParam(r, "tags"),
		DatacenterID: r.URL.Query().Get("datacenter_id"),
		NetworkID:    r.URL.Query().Get("network_id"),
	}
	devices, err := h.storage.ListDevices(filter)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request) {
	var device model.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}
	if device.Name == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Name is required")
		return
	}
	if err := h.storage.CreateDevice(&device); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, device)
}

func (h *Handler) getDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	device, err := h.storage.GetDevice(id)
	if err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			h.writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Device not found")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, device)
}

func (h *Handler) updateDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	device, err := h.storage.GetDevice(id)
	if err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			h.writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Device not found")
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
	if tags, ok := updates["tags"].([]any); ok {
		device.Tags = toStringSlice(tags)
	}
	if domains, ok := updates["domains"].([]any); ok {
		device.Domains = toStringSlice(domains)
	}
	if addresses, ok := updates["addresses"].([]any); ok {
		device.Addresses = toAddressSlice(addresses)
	}

	if err := h.storage.UpdateDevice(device); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, device)
}

func (h *Handler) deleteDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.storage.DeleteDevice(id); err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			h.writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Device not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) searchDevices(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Query parameter 'q' is required")
		return
	}
	devices, err := h.storage.SearchDevices(query)
	if err != nil {
		h.internalError(w, err)
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
