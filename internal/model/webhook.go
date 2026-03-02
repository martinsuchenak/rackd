package model

import "time"

// EventType represents a type of event that can trigger webhooks
type EventType string

const (
	// Device events
	EventTypeDeviceCreated  EventType = "device.created"
	EventTypeDeviceUpdated  EventType = "device.updated"
	EventTypeDeviceDeleted  EventType = "device.deleted"
	EventTypeDevicePromoted EventType = "device.promoted"

	// Network events
	EventTypeNetworkCreated EventType = "network.created"
	EventTypeNetworkUpdated EventType = "network.updated"
	EventTypeNetworkDeleted EventType = "network.deleted"

	// Discovery events
	EventTypeDiscoveryStarted  EventType = "discovery.started"
	EventTypeDiscoveryCompleted EventType = "discovery.completed"
	EventTypeDeviceDiscovered  EventType = "discovery.device_found"

	// Conflict events
	EventTypeConflictDetected EventType = "conflict.detected"
	EventTypeConflictResolved EventType = "conflict.resolved"

	// Pool events
	EventTypePoolUtilization EventType = "pool.utilization_high"
)

// AllEventTypes contains all available event types
var AllEventTypes = []EventType{
	EventTypeDeviceCreated,
	EventTypeDeviceUpdated,
	EventTypeDeviceDeleted,
	EventTypeDevicePromoted,
	EventTypeNetworkCreated,
	EventTypeNetworkUpdated,
	EventTypeNetworkDeleted,
	EventTypeDiscoveryStarted,
	EventTypeDiscoveryCompleted,
	EventTypeDeviceDiscovered,
	EventTypeConflictDetected,
	EventTypeConflictResolved,
	EventTypePoolUtilization,
}

// IsValid checks if the event type is valid
func (e EventType) IsValid() bool {
	for _, et := range AllEventTypes {
		if e == et {
			return true
		}
	}
	return false
}

// String returns the string representation
func (e EventType) String() string {
	return string(e)
}

// Webhook represents a webhook endpoint configuration
type Webhook struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	URL         string      `json:"url"`
	Secret      string      `json:"-"` // Used for HMAC signature, never exposed in API responses
	HasSecret   bool        `json:"has_secret"`       // Indicates whether a secret is configured
	Events      []EventType `json:"events"`
	Active      bool        `json:"active"`
	Description string      `json:"description,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	CreatedBy   string      `json:"created_by,omitempty"`
}

// WebhookDelivery represents a delivery attempt for a webhook
type WebhookDelivery struct {
	ID            string          `json:"id"`
	WebhookID     string          `json:"webhook_id"`
	EventType     EventType       `json:"event_type"`
	Payload       string          `json:"payload"`
	ResponseCode  int             `json:"response_code,omitempty"`
	ResponseBody  string          `json:"response_body,omitempty"`
	Error         string          `json:"error,omitempty"`
	Duration      int64           `json:"duration_ms"`
	Status        DeliveryStatus  `json:"status"`
	AttemptNumber int             `json:"attempt_number"`
	NextRetry     *time.Time      `json:"next_retry,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// DeliveryStatus represents the status of a webhook delivery
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusSuccess   DeliveryStatus = "success"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
	DeliveryStatusAbandoned DeliveryStatus = "abandoned"
)

// IsValid checks if the delivery status is valid
func (s DeliveryStatus) IsValid() bool {
	return s == DeliveryStatusPending ||
		s == DeliveryStatusSuccess ||
		s == DeliveryStatusFailed ||
		s == DeliveryStatusRetrying ||
		s == DeliveryStatusAbandoned
}

// WebhookFilter for querying webhooks
type WebhookFilter struct {
	Active *bool
	Events []EventType
}

// DeliveryFilter for querying deliveries
type DeliveryFilter struct {
	WebhookID string
	Status    DeliveryStatus
	EventType EventType
	After     *time.Time
	Before    *time.Time
	Limit     int
}

// Event represents an internal event to be dispatched
type Event struct {
	ID        string      `json:"id"`
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// EventPayloadDevice contains device event data
type EventPayloadDevice struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Hostname string       `json:"hostname,omitempty"`
	Status   DeviceStatus `json:"status,omitempty"`
	Changes  []string     `json:"changes,omitempty"` // For update events
}

// EventPayloadNetwork contains network event data
type EventPayloadNetwork struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Subnet string `json:"subnet"`
}

// EventPayloadDiscovery contains discovery event data
type EventPayloadDiscovery struct {
	NetworkID   string `json:"network_id,omitempty"`
	DevicesFound int   `json:"devices_found,omitempty"`
	Duration    int64  `json:"duration_ms,omitempty"`
}

// EventPayloadConflict contains conflict event data
type EventPayloadConflict struct {
	ID          string       `json:"id"`
	Type        string       `json:"type"`
	Description string       `json:"description"`
	DeviceIDs   []string     `json:"device_ids,omitempty"`
}

// EventPayloadPoolUtilization contains pool utilization event data
type EventPayloadPoolUtilization struct {
	PoolID      string  `json:"pool_id"`
	PoolName    string  `json:"pool_name"`
	NetworkID   string  `json:"network_id"`
	Utilization float64 `json:"utilization"`
	Threshold   float64 `json:"threshold"`
}

// CreateWebhookRequest represents a request to create a webhook
type CreateWebhookRequest struct {
	Name        string      `json:"name"`
	URL         string      `json:"url"`
	Secret      string      `json:"secret,omitempty"`
	Events      []EventType `json:"events"`
	Active      bool        `json:"active"`
	Description string      `json:"description,omitempty"`
}

// UpdateWebhookRequest represents a request to update a webhook
type UpdateWebhookRequest struct {
	Name        *string     `json:"name,omitempty"`
	URL         *string     `json:"url,omitempty"`
	Secret      *string     `json:"secret,omitempty"`
	Events      *[]EventType `json:"events,omitempty"`
	Active      *bool       `json:"active,omitempty"`
	Description *string     `json:"description,omitempty"`
}
