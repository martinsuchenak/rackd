package dns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	securl "github.com/martinsuchenak/rackd/internal/webhook"
)

// TechnitiumClient implements the Provider interface for Technitium DNS Server
type TechnitiumClient struct {
	endpoint string
	token    string
	client   *http.Client
}

// NewTechnitiumClient creates a new Technitium DNS client
func NewTechnitiumClient(endpoint, token string) *TechnitiumClient {
	// The client carries the provider API token, so it must not be
	// redirectable to arbitrary hosts and must not dial loopback,
	// link-local (cloud metadata) or private ranges. The secure client
	// enforces this at dial time and re-validates every redirect hop.
	return NewTechnitiumClientWithHTTPClient(endpoint, token, securl.NewSecureHTTPClient(30*time.Second))
}

// NewTechnitiumClientWithHTTPClient creates a Technitium DNS client that
// uses the supplied HTTP client. It exists so tests can inject a transport
// bound to a local test server; production code must use NewTechnitiumClient,
// whose client enforces the SSRF protections.
func NewTechnitiumClientWithHTTPClient(endpoint, token string, client *http.Client) *TechnitiumClient {
	// Normalize: the endpoint is the web console root (e.g. http://host:5380);
	// a trailing slash or a trailing "/api" is a common copy-paste mistake and
	// would make every request hit a non-existent path (404).
	endpoint = strings.TrimSuffix(endpoint, "/")
	endpoint = strings.TrimSuffix(endpoint, "/api")
	endpoint = strings.TrimSuffix(endpoint, "/")
	return &TechnitiumClient{
		endpoint: endpoint,
		token:    token,
		client:   client,
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
		TTL      uint32 `json:"ttl"`
		Disabled bool   `json:"disabled"`
		// Technitium nests record data in rData object
		RData struct {
			IPAddress         string `json:"ipAddress,omitempty"`  // A/AAAA records
			CNAME             string `json:"cname,omitempty"`      // CNAME records
			NameServer        string `json:"nameServer,omitempty"` // NS records
			Exchange          string `json:"exchange,omitempty"`   // MX records
			Text              string `json:"text,omitempty"`       // TXT records
			PtrName           string `json:"ptrName,omitempty"`    // PTR records
			PrimaryNameServer string `json:"primaryNameServer,omitempty"` // SOA records
		} `json:"rData"`
	} `json:"records"`
}

// doAPI executes an API call to the Technitium DNS server
func (c *TechnitiumClient) doAPI(ctx context.Context, method, path string, params url.Values, result interface{}) error {
	if params == nil {
		params = url.Values{}
	}

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
	// Authenticate via Authorization header rather than a token query
	// parameter: query strings leak into proxy/access logs and into
	// url.Error messages on transport failures.
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		// Strip the URL from wrapped transport errors; it can embed query
		// parameters (kept above for any future sensitive params).
		var ue *url.Error
		if errors.As(err, &ue) {
			ue.URL = ""
			return fmt.Errorf("request failed: %w", ue)
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Technitium < v15 only authenticates via the token query parameter and
	// ignores the Authorization header; depending on version an
	// unauthenticated call 401s or 404s. Retry once with the legacy
	// parameter so older servers keep working (header-only for v15+).
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		params.Set("token", c.token)
		fullURL := c.endpoint + path + "?" + params.Encode()
		if method == "POST" {
			req, err = http.NewRequestWithContext(ctx, "POST", fullURL, nil)
		} else {
			req, err = http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		}
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		resp, err = c.client.Do(req)
		if err != nil {
			var ue *url.Error
			if errors.As(err, &ue) {
				ue.URL = ""
				return fmt.Errorf("request failed: %w", ue)
			}
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()
	}

	// Check for non-2xx HTTP status before attempting JSON decode. Include a
	// body excerpt so operators can tell a reverse-proxy 404 (HTML) from a
	// Technitium error (JSON) and see the server's own message.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if len(body) > 0 {
			return fmt.Errorf("HTTP error: status %d %s (path %s, response: %s)", resp.StatusCode, resp.Status, path, strings.TrimSpace(string(body)))
		}
		return fmt.Errorf("HTTP error: status %d %s (path %s, empty response body)", resp.StatusCode, resp.Status, path)
	}

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
// Uses create-then-delete pattern to avoid a window where the record doesn't exist.
// Technitium allows duplicate records, so we add the new one first, then remove the old.
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

	// Create the new record first so there's no gap in DNS resolution
	if err := c.CreateRecord(ctx, zone, record); err != nil {
		return fmt.Errorf("failed to create new record during update: %w", err)
	}

	// Now delete the old record; if this fails, we have a duplicate but no data loss
	if err := c.DeleteRecord(ctx, zone, record.Name, record.Type); err != nil {
		// Log-worthy but not fatal — the new record is already in place
		// The old value may linger as a duplicate until next sync
	}

	return nil
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
				Name:  r.Name,
				Type:  r.Type,
				Value: extractRecordValue(r.RData.IPAddress, r.RData.CNAME, r.RData.NameServer, r.RData.Exchange, r.RData.Text, r.RData.PtrName, r.RData.PrimaryNameServer),
				TTL:   int(r.TTL),
			}, nil
		}
	}

	return nil, fmt.Errorf("record %s/%s not found in zone %s", name, rtype, zone)
}

// ListRecords lists all records in a zone
func (c *TechnitiumClient) ListRecords(ctx context.Context, zone string) ([]*Record, error) {
	params := url.Values{}
	params.Set("zone", zone)
	params.Set("domain", zone) // domain is required by the API
	params.Set("listZone", "true") // list all records in the zone, not just for the domain

	var resp recordsGetResponse
	if err := c.doAPI(ctx, "GET", "/api/zones/records/get", params, &resp); err != nil {
		return nil, err
	}

	records := make([]*Record, 0, len(resp.Records))
	for _, r := range resp.Records {
		records = append(records, &Record{
			Name:  r.Name,
			Type:  r.Type,
			Value: extractRecordValue(r.RData.IPAddress, r.RData.CNAME, r.RData.NameServer, r.RData.Exchange, r.RData.Text, r.RData.PtrName, r.RData.PrimaryNameServer),
			TTL:   int(r.TTL),
		})
	}

	return records, nil
}

// extractRecordValue extracts the value from a Technitium record response
// Technitium uses different field names depending on record type
func extractRecordValue(fields ...string) string {
	// Return the first non-empty field
	for _, f := range fields {
		if f != "" {
			return f
		}
	}
	return ""
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
	// The /api/status endpoint returns server info; we just need to verify the API responds
	// doAPI already checks that the outer status is "ok"
	var resp map[string]interface{}
	if err := c.doAPI(ctx, "GET", "/api/status", nil, &resp); err != nil {
		return err
	}

	return nil
}

