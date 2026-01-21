package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type startScanRequest struct {
	ScanType string `json:"scan_type"`
}

func (h *Handler) startScan(w http.ResponseWriter, r *http.Request) {
	networkID := r.PathValue("id")
	var req startScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.ScanType = model.ScanTypeQuick
	}
	if req.ScanType == "" {
		req.ScanType = model.ScanTypeQuick
	}
	if !isValidScanType(req.ScanType) {
		h.writeError(w, http.StatusBadRequest, "INVALID_TYPE", "scan_type must be quick, full, or deep")
		return
	}
	now := time.Now()
	scan := &model.DiscoveryScan{
		ID:        uuid.Must(uuid.NewV7()).String(),
		NetworkID: networkID,
		Status:    model.ScanStatusPending,
		ScanType:  req.ScanType,
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.storage.CreateDiscoveryScan(scan); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, scan)
}

func (h *Handler) listScans(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")
	scans, err := h.storage.ListDiscoveryScans(networkID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scans)
}

func (h *Handler) getScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scan, err := h.storage.GetDiscoveryScan(id)
	if err != nil {
		if errors.Is(err, storage.ErrScanNotFound) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Scan not found")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scan)
}

func (h *Handler) listDiscoveredDevices(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")
	devices, err := h.storage.ListDiscoveredDevices(networkID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}

type promoteRequest struct {
	Name         string `json:"name"`
	MakeModel    string `json:"make_model"`
	DatacenterID string `json:"datacenter_id"`
}

func (h *Handler) promoteDevice(w http.ResponseWriter, r *http.Request) {
	discoveredID := r.PathValue("id")
	var req promoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}
	discovered, err := h.storage.GetDiscoveredDevice(discoveredID)
	if err != nil {
		if errors.Is(err, storage.ErrDiscoveryNotFound) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Discovered device not found")
			return
		}
		h.internalError(w, err)
		return
	}
	now := time.Now()
	device := &model.Device{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Name:         req.Name,
		MakeModel:    req.MakeModel,
		DatacenterID: req.DatacenterID,
		Addresses:    []model.Address{{IP: discovered.IP, Type: "ipv4"}},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if device.Name == "" {
		device.Name = discovered.Hostname
	}
	if err := h.storage.CreateDevice(device); err != nil {
		h.internalError(w, err)
		return
	}
	if err := h.storage.PromoteDiscoveredDevice(discoveredID, device.ID); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, device)
}

func (h *Handler) listDiscoveryRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.storage.ListDiscoveryRules()
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rules)
}

type discoveryRuleRequest struct {
	NetworkID     string `json:"network_id"`
	Enabled       bool   `json:"enabled"`
	ScanType      string `json:"scan_type"`
	IntervalHours int    `json:"interval_hours"`
	ExcludeIPs    string `json:"exclude_ips"`
}

func (h *Handler) createDiscoveryRule(w http.ResponseWriter, r *http.Request) {
	var req discoveryRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}
	if req.NetworkID == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_FIELD", "network_id is required")
		return
	}
	now := time.Now()
	rule := &model.DiscoveryRule{
		ID:            uuid.Must(uuid.NewV7()).String(),
		NetworkID:     req.NetworkID,
		Enabled:       req.Enabled,
		ScanType:      req.ScanType,
		IntervalHours: req.IntervalHours,
		ExcludeIPs:    req.ExcludeIPs,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if rule.ScanType == "" {
		rule.ScanType = model.ScanTypeQuick
	}
	if rule.IntervalHours == 0 {
		rule.IntervalHours = 24
	}
	if err := h.storage.SaveDiscoveryRule(rule); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) getDiscoveryRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rule, err := h.storage.GetDiscoveryRule(id)
	if err != nil {
		if errors.Is(err, storage.ErrRuleNotFound) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Discovery rule not found")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) updateDiscoveryRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := h.storage.GetDiscoveryRule(id)
	if err != nil {
		if errors.Is(err, storage.ErrRuleNotFound) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Discovery rule not found")
			return
		}
		h.internalError(w, err)
		return
	}
	var req discoveryRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}
	existing.Enabled = req.Enabled
	if req.ScanType != "" {
		existing.ScanType = req.ScanType
	}
	if req.IntervalHours > 0 {
		existing.IntervalHours = req.IntervalHours
	}
	existing.ExcludeIPs = req.ExcludeIPs
	existing.UpdatedAt = time.Now()
	if err := h.storage.SaveDiscoveryRule(existing); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) deleteDiscoveryRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.storage.DeleteDiscoveryRule(id); err != nil {
		if errors.Is(err, storage.ErrRuleNotFound) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Discovery rule not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isValidScanType(t string) bool {
	return t == model.ScanTypeQuick || t == model.ScanTypeFull || t == model.ScanTypeDeep
}
