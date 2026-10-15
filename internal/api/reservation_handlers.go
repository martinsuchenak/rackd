package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func (h *Handler) listReservations(w http.ResponseWriter, r *http.Request) {
	filter := &model.ReservationFilter{}
	filter.Pagination = parseOptionalPagination(r)
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
	if reservations == nil {
		reservations = []model.Reservation{}
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

func (h *Handler) updateReservation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		model.UpdateReservationRequest
		ExpiresInDays int `json:"expires_in_days,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.invalidJSON(w)
		return
	}
	req := body.UpdateReservationRequest
	expiresAt, errs := reservationExpiry(req.ExpiresAt, body.ExpiresInDays)
	if len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}
	req.ExpiresAt = expiresAt

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
	w.WriteHeader(http.StatusNoContent)
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

// createReservationRequestWithDefaults is the create body: the expiry may be
// given as expires_at or expires_in_days.
type createReservationRequestWithDefaults struct {
	model.CreateReservationRequest
	ExpiresInDays int `json:"expires_in_days,omitempty"` // Days until expiration
}

// reservationExpiry resolves a reservation's expiry from either expires_at or
// expires_in_days (1 to 365); giving both is an error. Create and update take
// the same two forms.
func reservationExpiry(at *time.Time, days int) (*time.Time, ValidationErrors) {
	switch {
	case at != nil && days != 0:
		return nil, ValidationErrors{{Field: "expires_in_days", Message: "give expires_at or expires_in_days, not both"}}
	case days < 0 || days > 365:
		return nil, ValidationErrors{{Field: "expires_in_days", Message: "expires_in_days must be between 1 and 365"}}
	case days > 0:
		exp := time.Now().UTC().AddDate(0, 0, days)
		return &exp, nil
	}
	return at, nil
}

func (h *Handler) createReservationWithDefaults(w http.ResponseWriter, r *http.Request) {
	var req createReservationRequestWithDefaults
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}

	expiresAt, errs := reservationExpiry(req.ExpiresAt, req.ExpiresInDays)
	if len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}
	createReq := &req.CreateReservationRequest
	createReq.ExpiresAt = expiresAt

	reservation, err := h.svc.Reservations.Create(r.Context(), createReq)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, reservation)
}
