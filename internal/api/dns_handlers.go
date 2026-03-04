package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Provider endpoints

// listDNSProviders returns all DNS providers
func (h *Handler) listDNSProviders(w http.ResponseWriter, r *http.Request) {
	filter := &model.DNSProviderFilter{}
	if t := r.URL.Query().Get("type"); t != "" {
		filter.Type = model.DNSProviderType(t)
	}

	providers, err := h.svc.DNS.ListProviders(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Ensure we return an empty array, not null
	if providers == nil {
		providers = []model.DNSProviderConfig{}
	}
	h.writeJSON(w, http.StatusOK, providers)
}

// createDNSProvider creates a new DNS provider
func (h *Handler) createDNSProvider(w http.ResponseWriter, r *http.Request) {
	var req model.CreateDNSProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	provider, err := h.svc.DNS.CreateProvider(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, provider)
}

// getDNSProvider returns a single DNS provider by ID
func (h *Handler) getDNSProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	provider, err := h.svc.DNS.GetProvider(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, provider)
}

// updateDNSProvider updates an existing DNS provider
func (h *Handler) updateDNSProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	var req model.UpdateDNSProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	provider, err := h.svc.DNS.UpdateProvider(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, provider)
}

// deleteDNSProvider deletes a DNS provider
func (h *Handler) deleteDNSProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	if err := h.svc.DNS.DeleteProvider(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// testDNSProvider tests the connectivity to a DNS provider
func (h *Handler) testDNSProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	if err := h.svc.DNS.TestProvider(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// listDNSProviderZones lists zones for a provider
func (h *Handler) listDNSProviderZones(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	filter := &model.DNSZoneFilter{
		ProviderID: id,
	}

	zones, err := h.svc.DNS.ListZones(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Ensure we return an empty array, not null
	if zones == nil {
		zones = []model.DNSZone{}
	}
	h.writeJSON(w, http.StatusOK, zones)
}

// Zone endpoints

// listDNSZones returns all DNS zones
func (h *Handler) listDNSZones(w http.ResponseWriter, r *http.Request) {
	filter := &model.DNSZoneFilter{}
	if pid := r.URL.Query().Get("provider_id"); pid != "" {
		filter.ProviderID = pid
	}
	if nid := r.URL.Query().Get("network_id"); nid != "" {
		filter.NetworkID = &nid
	}
	if as := r.URL.Query().Get("auto_sync"); as != "" {
		val := as == "true"
		filter.AutoSync = &val
	}

	zones, err := h.svc.DNS.ListZones(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Ensure we return an empty array, not null
	if zones == nil {
		zones = []model.DNSZone{}
	}
	h.writeJSON(w, http.StatusOK, zones)
}

// createDNSZone creates a new DNS zone
func (h *Handler) createDNSZone(w http.ResponseWriter, r *http.Request) {
	var req model.CreateDNSZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	zone, err := h.svc.DNS.CreateZone(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, zone)
}

// getDNSZone returns a single DNS zone by ID
func (h *Handler) getDNSZone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	zone, err := h.svc.DNS.GetZone(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, zone)
}

// updateDNSZone updates an existing DNS zone
func (h *Handler) updateDNSZone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	var req model.UpdateDNSZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	zone, err := h.svc.DNS.UpdateZone(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, zone)
}

// deleteDNSZone deletes a DNS zone
func (h *Handler) deleteDNSZone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	if err := h.svc.DNS.DeleteZone(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// syncDNSZone syncs all pending records in a zone to the DNS provider
func (h *Handler) syncDNSZone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	result, err := h.svc.DNS.SyncZone(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

// importDNSZone imports all records from a DNS provider zone
func (h *Handler) importDNSZone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	result, err := h.svc.DNS.ImportFromDNS(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

// listDNSZoneRecords lists records in a zone
func (h *Handler) listDNSZoneRecords(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	filter := &model.DNSRecordFilter{
		ZoneID: id,
	}
	if did := r.URL.Query().Get("device_id"); did != "" {
		filter.DeviceID = &did
	}
	if t := r.URL.Query().Get("type"); t != "" {
		filter.Type = t
	}
	if ss := r.URL.Query().Get("sync_status"); ss != "" {
		status := model.RecordSyncStatus(ss)
		filter.SyncStatus = &status
	}
	if ls := r.URL.Query().Get("link_status"); ls != "" {
		filter.LinkStatus = &ls
	}

	records, err := h.svc.DNS.ListRecords(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Ensure we return an empty array, not null
	if records == nil {
		records = []model.DNSRecord{}
	}
	h.writeJSON(w, http.StatusOK, records)
}

// Record endpoints

// getDNSRecord returns a single DNS record by ID
func (h *Handler) getDNSRecord(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	record, err := h.svc.DNS.GetRecord(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, record)
}

// updateDNSRecord updates an existing DNS record
func (h *Handler) updateDNSRecord(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	var req model.UpdateDNSRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	record, err := h.svc.DNS.UpdateRecord(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, record)
}

// deleteDNSRecord deletes a DNS record
func (h *Handler) deleteDNSRecord(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	if err := h.svc.DNS.DeleteRecord(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// linkDNSRecord links an unlinked DNS record to a device
func (h *Handler) linkDNSRecord(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	var req model.LinkDNSRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	record, err := h.svc.DNS.LinkRecord(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, record)
}

// promoteDNSRecord creates a new device from an unlinked DNS record
func (h *Handler) promoteDNSRecord(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	var req model.PromoteDNSRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	record, err := h.svc.DNS.PromoteRecord(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, record)
}


