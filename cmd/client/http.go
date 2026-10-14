package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

// ErrServerTooOld is returned when the server answers an /api/v1 request with
// its web UI, which is what releases before the versioned API do.
var ErrServerTooOld = errors.New("the rackd server does not serve /api/v1; upgrade it to the same version as this CLI")

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
		if mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); mediaType == "text/html" {
			_ = resp.Body.Close()
			return nil, ErrServerTooOld
		}
	}
	return resp, nil
}
