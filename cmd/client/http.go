package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

// ErrNotV1Response is returned when an /api/v1 request gets an answer a rackd
// server with the versioned API never sends: a successful HTML page, or a 405
// that is not JSON. Older servers answer with their web UI (GET) or a plain-text
// 405 (other methods); a proxy in front of rackd may answer with a login page.
var ErrNotV1Response = errors.New("the server did not answer like a rackd /api/v1 server")

type Client struct {
	serverURL  string
	token      string
	httpClient *http.Client
}

func NewClient(cfg *Config) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !cfg.VerifySSL {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		serverURL: cfg.ServerURL,
		token:     cfg.Token,
		httpClient: &http.Client{
			Timeout:   cfg.GetTimeout(),
			Transport: transport,
		},
	}
}

func (c *Client) DoRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.serverURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(path, "/api/v1/") {
		mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		successHTML := resp.StatusCode < 300 && mediaType == "text/html"
		plain405 := resp.StatusCode == http.StatusMethodNotAllowed && mediaType != "application/json"
		if successHTML || plain405 {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("%w (%s %s returned %d %s): the server may be older than this CLI and need upgrading, or a proxy in front of it answered instead",
				ErrNotV1Response, method, path, resp.StatusCode, mediaType)
		}
	}
	return resp, nil
}
