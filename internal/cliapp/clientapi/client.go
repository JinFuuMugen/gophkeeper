package clientapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

type LoginResponse struct {
	Token      string `json:"token"`
	KDFSaltB64 string `json:"kdf_salt_b64"`
}

func (c *Client) Register(login, password string) (string, error) {
	reqBody := map[string]string{"login": login, "password": password}
	var resp struct {
		UserID string `json:"user_id"`
	}
	if err := c.doJSON(http.MethodPost, "/register", "", reqBody, &resp); err != nil {
		return "", err
	}
	return resp.UserID, nil
}

func (c *Client) Login(login, password string) (LoginResponse, error) {
	reqBody := map[string]string{"login": login, "password": password}
	var resp LoginResponse
	if err := c.doJSON(http.MethodPost, "/login", "", reqBody, &resp); err != nil {
		return LoginResponse{}, err
	}
	return resp, nil
}

type UpsertItemRequest struct {
	ID           string `json:"id,omitempty"`
	Type         string `json:"type"`
	EncryptedB64 string `json:"encrypted_b64,omitempty"`
	Metadata     string `json:"metadata,omitempty"`
	Version      int64  `json:"version,omitempty"`
	Deleted      bool   `json:"deleted,omitempty"`
}

type Item struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	EncryptedB64 string    `json:"encrypted_b64,omitempty"`
	Metadata     string    `json:"metadata"`
	Version      int64     `json:"version"`
	Deleted      bool      `json:"deleted"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (c *Client) UpsertItem(req UpsertItemRequest) (Item, error) {
	var out Item
	if err := c.doJSON(http.MethodPost, "/items/", c.token, req, &out); err != nil {
		return Item{}, err
	}
	return out, nil
}

func (c *Client) ListItems() ([]Item, error) {
	var out []Item
	if err := c.doJSON(http.MethodGet, "/items/", c.token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SyncItems(sinceRFC3339Nano string) ([]Item, error) {
	path := "/items/sync?since=" + sinceRFC3339Nano
	var out []Item
	if err := c.doJSON(http.MethodGet, path, c.token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) doJSON(method, path, bearer string, reqBody any, out any) error {
	var body io.Reader
	if reqBody != nil && method != http.MethodGet {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if reqBody != nil && method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(b, &e) == nil && e.Error != "" {
			return fmt.Errorf("server %s %s: %s", method, path, e.Error)
		}
		return fmt.Errorf("server %s %s: %s", method, path, strings.TrimSpace(string(b)))
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
