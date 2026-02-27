package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listReservations(w http.ResponseWriter, r *http.Request) {
	filter := &model.ReservationFilter{}
	if poolID := r.URL.Query().Get("pool_id"); poolID != "" {
		filter.PoolID = poolID
	}
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		filter.Status = model.ReservationStatus(statusStr)
	}
	if reservedBy := r.URL.Query().Get("reserved_by"); reservedBy != "" {
		filter.ReservedBy = reservedBy
	}
	if ip := r.URL.Query().Get("ip"); ip != "" {
		filter.IPAddress = ip
	}

	reservations, err := h.svc.Reservations.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, reservations)
}

func (h *Handler) getReservation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	reservation, err := h.svc.Reservations.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, reservation)
}

func (h *Handler) createReservation(w http.ResponseWriter, r *http.Request) {
	var req model.CreateReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	reservation, err := h.svc.Reservations.Create(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, reservation)
}

func (h *Handler) updateReservation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req model.UpdateReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	reservation, err := h.svc.Reservations.Update(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, reservation)
}

func (h *Handler) deleteReservation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Reservations.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message": "Reservation deleted successfully",
	})
}

func (h *Handler) releaseReservation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Reservations.Release(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message": "Reservation released successfully",
	})
}

func (h *Handler) listPoolReservations(w http.ResponseWriter, r *http.Request) {
	poolID := r.PathValue("id")

	reservations, err := h.svc.Reservations.GetByPool(r.Context(), poolID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, reservations)
}

func (h *Handler) getReservationByIP(w http.ResponseWriter, r *http.Request) {
	poolID := r.PathValue("poolId")
	ip := r.PathValue("ip")

	reservation, err := h.svc.Reservations.GetByIP(r.Context(), poolID, ip)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, reservation)
}

// createReservationRequestWithDefaults is used for parsing requests with optional expires_in_days field
type createReservationRequestWithDefaults struct {
	PoolID        string `json:"pool_id"`
	IPAddress     string `json:"ip_address,omitempty"`
	Hostname      string `json:"hostname,omitempty"`
	Purpose       string `json:"purpose,omitempty"`
	ExpiresInDays int    `json:"expires_in_days,omitempty"` // Days until expiration
	Notes         string `json:"notes,omitempty"`
}

func (h *Handler) createReservationWithDefaults(w http.ResponseWriter, r *http.Request) {
	var req createReservationRequestWithDefaults
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	// Convert expires_in_days to *time.Time
	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		exp := time.Now().UTC().AddDate(0, 0, req.ExpiresInDays)
		expiresAt = &exp
	}

	createReq := &model.CreateReservationRequest{
		PoolID:    req.PoolID,
		IPAddress: req.IPAddress,
		Hostname:  req.Hostname,
		Purpose:   req.Purpose,
		ExpiresAt: expiresAt,
		Notes:     req.Notes,
	}

	reservation, err := h.svc.Reservations.Create(r.Context(), createReq)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, reservation)
}
