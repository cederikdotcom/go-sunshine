package sunshine

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps the Sunshine REST API.
type Client struct {
	BaseURL  string
	Username string
	Password string
	HTTP     *http.Client
}

// NewClient creates a Sunshine API client. The baseURL is typically
// https://localhost:47990. TLS verification is disabled by default
// because Sunshine uses a self-signed certificate.
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		HTTP: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // localhost self-signed cert
			},
		},
	}
}

// appsResponse is the JSON envelope returned by GET /api/apps.
type appsResponse struct {
	Apps []App `json:"apps"`
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
	return c.HTTP.Do(req)
}

func (c *Client) doJSON(method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// ListApps returns the currently registered Sunshine applications.
func (c *Client) ListApps() ([]App, error) {
	data, err := c.doJSON("GET", "/api/apps", nil)
	if err != nil {
		return nil, err
	}
	var result appsResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing apps: %w", err)
	}
	return result.Apps, nil
}

// RegisterApp adds a new application to Sunshine (index should be -1).
func (c *Client) RegisterApp(app App) error {
	_, err := c.doJSON("POST", "/api/apps", app)
	return err
}

// UpdateApp updates an existing application (index >= 0).
func (c *Client) UpdateApp(app App) error {
	_, err := c.doJSON("POST", "/api/apps", app)
	return err
}

// DeleteApp removes an application by its index.
func (c *Client) DeleteApp(index int) error {
	_, err := c.doJSON("DELETE", fmt.Sprintf("/api/apps/%d", index), nil)
	return err
}

// CloseRunningApp closes the currently running streamed application.
func (c *Client) CloseRunningApp() error {
	_, err := c.doJSON("POST", "/api/apps/close", struct{}{})
	return err
}

// ListClients returns paired Moonlight clients.
func (c *Client) ListClients() ([]ClientInfo, error) {
	data, err := c.doJSON("GET", "/api/clients/list", nil)
	if err != nil {
		return nil, err
	}
	var clients []ClientInfo
	if err := json.Unmarshal(data, &clients); err != nil {
		return nil, fmt.Errorf("parsing clients: %w", err)
	}
	return clients, nil
}

// UnpairClient removes a specific paired client by UUID.
func (c *Client) UnpairClient(uuid string) error {
	_, err := c.doJSON("POST", "/api/clients/unpair", map[string]string{"uuid": uuid})
	return err
}

// UnpairAll removes all paired clients.
func (c *Client) UnpairAll() error {
	_, err := c.doJSON("POST", "/api/clients/unpair-all", nil)
	return err
}

// Restart restarts the Sunshine service.
func (c *Client) Restart() error {
	_, err := c.doJSON("POST", "/api/restart", nil)
	return err
}

// GetConfig returns the Sunshine configuration as key-value pairs.
func (c *Client) GetConfig() (map[string]string, error) {
	data, err := c.doJSON("GET", "/api/config", nil)
	if err != nil {
		return nil, err
	}
	var cfg map[string]string
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// SetConfig updates Sunshine configuration key-value pairs.
func (c *Client) SetConfig(cfg map[string]string) error {
	_, err := c.doJSON("POST", "/api/config", cfg)
	return err
}

// GetLogs returns the Sunshine log output.
func (c *Client) GetLogs() (string, error) {
	data, err := c.doJSON("GET", "/api/logs", nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
