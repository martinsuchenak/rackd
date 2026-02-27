package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

// listWebhooks returns all webhooks
func (h *Handler) listWebhooks(w http.ResponseWriter, r *http.Request) {
	filter := &model.WebhookFilter{}
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		active := activeStr == "true"
		filter.Active = &active
	}

	webhooks, err := h.svc.Webhooks.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, webhooks)
}

// getWebhook returns a single webhook by ID
func (h *Handler) getWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	webhook, err := h.svc.Webhooks.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, webhook)
}

// createWebhook creates a new webhook
func (h *Handler) createWebhook(w http.ResponseWriter, r *http.Request) {
	var req model.CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	webhook, err := h.svc.Webhooks.Create(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, webhook)
}

// updateWebhook updates an existing webhook
func (h *Handler) updateWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req model.UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	webhook, err := h.svc.Webhooks.Update(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, webhook)
}

// deleteWebhook deletes a webhook
func (h *Handler) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Webhooks.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message": "Webhook deleted successfully",
	})
}

// pingWebhook sends a test event to a webhook
func (h *Handler) pingWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	delivery, err := h.svc.Webhooks.Ping(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message":  "Webhook ping sent successfully",
		"delivery": delivery,
	})
}

// listWebhookDeliveries returns delivery records for a webhook
func (h *Handler) listWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	webhookID := r.PathValue("id")

	filter := &model.DeliveryFilter{
		WebhookID: webhookID,
	}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		filter.Status = model.DeliveryStatus(statusStr)
	}
	if eventTypeStr := r.URL.Query().Get("event_type"); eventTypeStr != "" {
		filter.EventType = model.EventType(eventTypeStr)
	}

	deliveries, err := h.svc.Webhooks.ListDeliveries(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, deliveries)
}

// getWebhookDelivery returns a single delivery record
func (h *Handler) getWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("deliveryId")

	delivery, err := h.svc.Webhooks.GetDelivery(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, delivery)
}

// getEventTypes returns all available event types
func (h *Handler) getEventTypes(w http.ResponseWriter, r *http.Request) {
	eventTypes := make([]map[string]string, len(model.AllEventTypes))
	for i, et := range model.AllEventTypes {
		eventTypes[i] = map[string]string{
			"value": string(et),
			"label": getEventLabel(et),
		}
	}
	h.writeJSON(w, http.StatusOK, eventTypes)
}

// getEventLabel returns a human-readable label for an event type
func getEventLabel(et model.EventType) string {
	labels := map[model.EventType]string{
		model.EventTypeDeviceCreated:     "Device Created",
		model.EventTypeDeviceUpdated:     "Device Updated",
		model.EventTypeDeviceDeleted:     "Device Deleted",
		model.EventTypeDevicePromoted:    "Device Promoted",
		model.EventTypeNetworkCreated:    "Network Created",
		model.EventTypeNetworkUpdated:    "Network Updated",
		model.EventTypeNetworkDeleted:    "Network Deleted",
		model.EventTypeDiscoveryStarted:  "Discovery Started",
		model.EventTypeDiscoveryCompleted: "Discovery Completed",
		model.EventTypeDeviceDiscovered:  "Device Discovered",
		model.EventTypeConflictDetected:  "Conflict Detected",
		model.EventTypeConflictResolved:  "Conflict Resolved",
		model.EventTypePoolUtilization:   "Pool Utilization High",
	}
	if label, ok := labels[et]; ok {
		return label
	}
	return string(et)
}
