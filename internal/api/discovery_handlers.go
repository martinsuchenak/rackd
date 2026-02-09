package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
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

	if h.svc != nil && h.svc.Discovery != nil {
		scan, err := h.svc.Discovery.StartScan(r.Context(), networkID, req.ScanType)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		log.Info("Scan started successfully", "scan_id", scan.ID, "network_id", networkID, "status", scan.Status)
		h.writeJSON(w, http.StatusAccepted, scan)
		return
	}

	if h.scanner == nil {
		h.writeError(w, http.StatusInternalServerError, "SCANNER_NOT_INITIALIZED", "Discovery scanner not initialized")
		return
	}

	network, err := h.store.GetNetwork(networkID)
	if err != nil {
		log.Error("Network not found for scan", "network_id", networkID, "error", err)
		h.writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found")
		return
	}

	log.Info("Scan request received", "network_id", networkID, "network_name", network.Name, "subnet", network.Subnet, "scan_type", req.ScanType)

	scan, err := h.scanner.Scan(h.auditContext(r), network, req.ScanType)
	if err != nil {
		log.Error("Failed to start scan", "network_id", networkID, "network_name", network.Name, "error", err)
		h.internalError(w, err)
		return
	}
	log.Info("Scan started successfully", "scan_id", scan.ID, "network_id", networkID, "status", scan.Status)
	h.writeJSON(w, http.StatusAccepted, scan)
}

func (h *Handler) listScans(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")

	if h.svc != nil && h.svc.Discovery != nil {
		scans, err := h.svc.Discovery.ListScans(r.Context(), networkID)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, scans)
		return
	}

	scans, err := h.store.ListDiscoveryScans(networkID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scans)
}

func (h *Handler) getScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Discovery != nil {
		scan, err := h.svc.Discovery.GetScan(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, scan)
		return
	}

	scan, err := h.store.GetDiscoveryScan(id)
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

func (h *Handler) cancelScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Discovery != nil {
		if err := h.svc.Discovery.CancelScan(r.Context(), id); err != nil {
			h.handleServiceError(w, err)
			return
		}
		// Return updated scan status
		scan, err := h.svc.Discovery.GetScan(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, scan)
		return
	}

	if h.scanner == nil {
		h.writeError(w, http.StatusInternalServerError, "SCANNER_NOT_INITIALIZED", "Discovery scanner not initialized")
		return
	}

	err := h.scanner.CancelScan(id)
	if err != nil {
		if errors.Is(err, discovery.ErrScanNotFound) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Scan not found")
			return
		}
		if errors.Is(err, discovery.ErrScanNotRunning) {
			h.writeError(w, http.StatusBadRequest, "NOT_RUNNING", "Scan is not running or pending")
			return
		}
		h.internalError(w, err)
		return
	}

	// Return updated scan status
	scan, err := h.scanner.GetScanStatus(id)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scan)
}

func (h *Handler) listDiscoveredDevices(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")

	if h.svc != nil && h.svc.Discovery != nil {
		devices, err := h.svc.Discovery.ListDevices(r.Context(), networkID)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, devices)
		return
	}

	devices, err := h.store.ListDiscoveredDevices(networkID)
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

	if h.svc != nil && h.svc.Discovery != nil {
		now := time.Now()
		device := &model.Device{
			Name:         req.Name,
			MakeModel:    req.MakeModel,
			DatacenterID: req.DatacenterID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if device.Name == "" {
			h.writeError(w, http.StatusBadRequest, "MISSING_FIELD", "name is required")
			return
		}

		promoted, err := h.svc.Discovery.PromoteDevice(r.Context(), discoveredID, device)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusCreated, promoted)
		return
	}

	discovered, err := h.store.GetDiscoveredDevice(discoveredID)
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
	if err := h.store.CreateDevice(h.auditContext(r), device); err != nil {
		h.internalError(w, err)
		return
	}
	if err := h.store.PromoteDiscoveredDevice(h.auditContext(r), discoveredID, device.ID); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, device)
}

func (h *Handler) deleteDiscoveredDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Discovery != nil {
		if err := h.svc.Discovery.DeleteDevice(r.Context(), id); err != nil {
			h.handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.store.DeleteDiscoveredDevice(h.auditContext(r), id); err != nil {
		if errors.Is(err, storage.ErrDiscoveryNotFound) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Discovered device not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteDiscoveryScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Discovery != nil {
		if err := h.svc.Discovery.DeleteScan(r.Context(), id); err != nil {
			h.handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.store.DeleteDiscoveryScan(h.auditContext(r), id); err != nil {
		if errors.Is(err, storage.ErrScanNotFound) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Scan not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteDiscoveredDevicesByNetwork(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")

	if h.svc != nil && h.svc.Discovery != nil {
		if err := h.svc.Discovery.DeleteDevicesByNetwork(r.Context(), networkID); err != nil {
			h.handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.store.DeleteDiscoveredDevicesByNetwork(h.auditContext(r), networkID); err != nil {
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listDiscoveryRules(w http.ResponseWriter, r *http.Request) {
	if h.svc != nil && h.svc.Discovery != nil {
		rules, err := h.svc.Discovery.ListRules(r.Context())
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, rules)
		return
	}

	rules, err := h.store.ListDiscoveryRules()
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

	if h.svc != nil && h.svc.Discovery != nil {
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
	if err := h.store.SaveDiscoveryRule(h.auditContext(r), rule); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) getDiscoveryRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Discovery != nil {
		rule, err := h.svc.Discovery.GetRule(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, rule)
		return
	}

	rule, err := h.store.GetDiscoveryRule(id)
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

	if h.svc != nil && h.svc.Discovery != nil {
		existing, err := h.svc.Discovery.GetRule(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
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
		if err := h.svc.Discovery.UpdateRule(r.Context(), existing); err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, existing)
		return
	}

	existing, err := h.store.GetDiscoveryRule(id)
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
	if err := h.store.SaveDiscoveryRule(h.auditContext(r), existing); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) deleteDiscoveryRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Discovery != nil {
		if err := h.svc.Discovery.DeleteRule(r.Context(), id); err != nil {
			h.handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.store.DeleteDiscoveryRule(h.auditContext(r), id); err != nil {
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
