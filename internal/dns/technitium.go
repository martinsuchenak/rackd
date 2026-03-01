package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TechnitiumClient implements the Provider interface for Technitium DNS Server
type TechnitiumClient struct {
	endpoint string
	token    string
	client   *http.Client
}

// NewTechnitiumClient creates a new Technitium DNS client
func NewTechnitiumClient(endpoint, token string) *TechnitiumClient {
	return &TechnitiumClient{
		endpoint: strings.TrimSuffix(endpoint, "/"),
		token:    token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider type name
func (c *TechnitiumClient) Name() string {
	return "technitium"
}

// apiResponse represents the standard Technitium API response format
type apiResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"errorMessage,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
}

// zoneListResponse represents the response from zones/list API
type zoneListResponse struct {
	Zones []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"zones"`
}

// recordsGetResponse represents the response from zones/records/get API
type recordsGetResponse struct {
	Records []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Data     string `json:"data"`
		TTL      uint32 `json:"ttl"`
		Priority uint16 `json:"priority,omitempty"`
	} `json:"records"`
}

// statusResponse represents the response from status API
type statusResponse struct {
	Status string `json:"status"`
}

// doAPI executes an API call to the Technitium DNS server
func (c *TechnitiumClient) doAPI(ctx context.Context, method, path string, params url.Values, result interface{}) error {
	// Build URL with token
	if params == nil {
		params = url.Values{}
	}
	params.Set("token", c.token)

	fullURL := c.endpoint + path + "?" + params.Encode()

	var req *http.Request
	var err error
	if method == "POST" {
		req, err = http.NewRequestWithContext(ctx, "POST", fullURL, nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	}
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Status != "ok" {
		if apiResp.Message != "" {
			return fmt.Errorf("API error: %s", apiResp.Message)
		}
		return fmt.Errorf("API error: status %s", apiResp.Status)
	}

	if result != nil && apiResp.Response != nil {
		if err := json.Unmarshal(apiResp.Response, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// CreateRecord creates a new DNS record in the zone
func (c *TechnitiumClient) CreateRecord(ctx context.Context, zone string, record *Record) error {
	params := url.Values{}
	params.Set("zone", zone)
	params.Set("domain", record.Name)
	params.Set("type", record.Type)
	params.Set("value", record.Value)
	if record.TTL > 0 {
		params.Set("ttl", fmt.Sprintf("%d", record.TTL))
	}
	if record.Priority != nil {
		params.Set("priority", fmt.Sprintf("%d", *record.Priority))
	}

	return c.doAPI(ctx, "POST", "/api/records/add", params, nil)
}

// UpdateRecord updates an existing DNS record in the zone
// Note: Technitium API uses delete + add pattern for updates, so we implement that
func (c *TechnitiumClient) UpdateRecord(ctx context.Context, zone string, record *Record) error {
	// First, get the existing record to check if it exists
	existing, err := c.GetRecord(ctx, zone, record.Name, record.Type)
	if err != nil {
		// Record doesn't exist, just create it
		return c.CreateRecord(ctx, zone, record)
	}

	// If the value is the same, no update needed
	if existing.Value == record.Value {
		return nil
	}

	// Delete the old record and create a new one
	if err := c.DeleteRecord(ctx, zone, record.Name, record.Type); err != nil {
		return fmt.Errorf("failed to delete old record during update: %w", err)
	}

	return c.CreateRecord(ctx, zone, record)
}

// DeleteRecord deletes a DNS record from the zone
func (c *TechnitiumClient) DeleteRecord(ctx context.Context, zone string, name string, rtype string) error {
	params := url.Values{}
	params.Set("zone", zone)
	params.Set("domain", name)
	params.Set("type", rtype)

	return c.doAPI(ctx, "POST", "/api/records/delete", params, nil)
}

// GetRecord retrieves a specific record from the zone
func (c *TechnitiumClient) GetRecord(ctx context.Context, zone string, name string, rtype string) (*Record, error) {
	params := url.Values{}
	params.Set("zone", zone)
	params.Set("domain", name)
	params.Set("type", rtype)

	var resp recordsGetResponse
	if err := c.doAPI(ctx, "GET", "/api/zones/records/get", params, &resp); err != nil {
		return nil, err
	}

	// Filter for the exact record we're looking for
	for _, r := range resp.Records {
		if r.Type == rtype && r.Name == name {
			return &Record{
				Name:     r.Name,
				Type:     r.Type,
				Value:    r.Data,
				TTL:      int(r.TTL),
				Priority: priorityPtr(r.Priority),
			}, nil
		}
	}

	return nil, fmt.Errorf("record %s/%s not found in zone %s", name, rtype, zone)
}

// ListRecords lists all records in a zone
func (c *TechnitiumClient) ListRecords(ctx context.Context, zone string) ([]*Record, error) {
	params := url.Values{}
	params.Set("zone", zone)

	var resp recordsGetResponse
	if err := c.doAPI(ctx, "GET", "/api/zones/records/get", params, &resp); err != nil {
		return nil, err
	}

	records := make([]*Record, 0, len(resp.Records))
	for _, r := range resp.Records {
		records = append(records, &Record{
			Name:     r.Name,
			Type:     r.Type,
			Value:    r.Data,
			TTL:      int(r.TTL),
			Priority: priorityPtr(r.Priority),
		})
	}

	return records, nil
}

// ListZones lists all available zones on the server
func (c *TechnitiumClient) ListZones(ctx context.Context) ([]string, error) {
	var resp zoneListResponse
	if err := c.doAPI(ctx, "GET", "/api/zones/list", nil, &resp); err != nil {
		return nil, err
	}

	zones := make([]string, 0, len(resp.Zones))
	for _, z := range resp.Zones {
		zones = append(zones, z.Name)
	}

	return zones, nil
}

// ZoneExists checks if a zone exists on the server
func (c *TechnitiumClient) ZoneExists(ctx context.Context, zone string) (bool, error) {
	zones, err := c.ListZones(ctx)
	if err != nil {
		return false, err
	}

	for _, z := range zones {
		if z == zone {
			return true, nil
		}
	}

	return false, nil
}

// HealthCheck verifies connectivity to the Technitium DNS server
func (c *TechnitiumClient) HealthCheck(ctx context.Context) error {
	var resp statusResponse
	if err := c.doAPI(ctx, "GET", "/api/status", nil, &resp); err != nil {
		return err
	}

	if resp.Status != "ok" {
		return fmt.Errorf("server status is not ok: %s", resp.Status)
	}

	return nil
}

// priorityPtr returns a pointer to the priority value, or nil if zero
func priorityPtr(p uint16) *int {
	if p == 0 {
		return nil
	}
	i := int(p)
	return &i
}
