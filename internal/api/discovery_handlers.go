package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
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
		h.badRequest(w, "scan_type must be quick, full, or deep")
		return
	}

	scan, err := h.svc.Discovery.StartScan(r.Context(), networkID, req.ScanType)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	log.Info("Scan started successfully", "scan_id", scan.ID, "network_id", networkID, "status", scan.Status)
	h.writeJSON(w, http.StatusAccepted, scan)
}

func (h *Handler) listScans(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")

	scans, err := h.svc.Discovery.ListScans(r.Context(), networkID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scans)
}

func (h *Handler) getScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	scan, err := h.svc.Discovery.GetScan(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scan)
}

func (h *Handler) cancelScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Discovery.CancelScan(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	scan, err := h.svc.Discovery.GetScan(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scan)
}

func (h *Handler) listDiscoveredDevices(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")

	devices, err := h.svc.Discovery.ListDevices(r.Context(), networkID)
	if err != nil {
		h.handleServiceError(w, err)
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
		h.invalidJSON(w)
		return
	}

	now := time.Now()
	device := &model.Device{
		Name:         req.Name,
		MakeModel:    req.MakeModel,
		DatacenterID: req.DatacenterID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if device.Name == "" {
		h.badRequest(w, "name is required")
		return
	}

	promoted, err := h.svc.Discovery.PromoteDevice(r.Context(), discoveredID, device)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, promoted)
}

func (h *Handler) deleteDiscoveredDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Discovery.DeleteDevice(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteDiscoveryScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Discovery.DeleteScan(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteDiscoveredDevicesByNetwork(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")

	if err := h.svc.Discovery.DeleteDevicesByNetwork(r.Context(), networkID); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listDiscoveryRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.svc.Discovery.ListRules(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
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
		h.invalidJSON(w)
		return
	}
	if req.NetworkID == "" {
		h.badRequest(w, "network_id is required")
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
	if err := h.svc.Discovery.CreateRule(r.Context(), rule); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) getDiscoveryRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	rule, err := h.svc.Discovery.GetRule(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) updateDiscoveryRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := h.svc.Discovery.GetRule(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	var req discoveryRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
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
	if err := h.svc.Discovery.UpdateRule(r.Context(), existing); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) deleteDiscoveryRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Discovery.DeleteRule(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isValidScanType(t string) bool {
	return t == model.ScanTypeQuick || t == model.ScanTypeFull || t == model.ScanTypeDeep
}
