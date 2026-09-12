// Package apiclient is a minimal HTTP client for the CommitIn backend.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client calls the CommitIn backend API.
type Client struct {
	baseURL string
	http    *http.Client
}

// New builds a Client for the backend at baseURL (e.g. "http://localhost:8080").
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// User mirrors the JSON shape the backend returns for an account.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthResponse is returned by both Signup and Login.
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Signup creates a new account and returns its session token.
func (c *Client) Signup(ctx context.Context, email, username, password string) (*AuthResponse, error) {
	var out AuthResponse
	req := map[string]string{"email": email, "username": username, "password": password}
	if err := c.postJSON(ctx, "/signup", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Login authenticates and returns a fresh session token.
func (c *Client) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	var out AuthResponse
	req := map[string]string{"email": email, "password": password}
	if err := c.postJSON(ctx, "/login", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) postJSON(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach CommitIn backend at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return apiError(resp)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// apiError turns a non-2xx response into an error, preferring the backend's
// {"error": "..."} body when present.
func apiError(resp *http.Response) error {
	var body struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Error != "" {
		return fmt.Errorf("%s", body.Error)
	}
	return fmt.Errorf("request failed: %s", resp.Status)
}
