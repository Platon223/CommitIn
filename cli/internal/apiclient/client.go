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

// Logout revokes the session behind token on the backend.
func (c *Client) Logout(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/logout", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach CommitIn backend at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return apiError(resp)
	}
	return nil
}

// SubmitScore records one judged commit-message attempt. attemptID is an
// opaque per-attempt identifier, not a git commit hash: the commit-msg hook
// runs before any commit object exists, so there's nothing to hash yet, and
// a rejected attempt never becomes a commit at all.
func (c *Client) SubmitScore(ctx context.Context, token string, score int, attemptID, repoName string) error {
	req := map[string]any{"score": score, "attempt_id": attemptID, "repo_name": repoName}
	return c.postAuthJSON(ctx, "/scores", token, req, nil)
}

// LeaderboardEntry mirrors one ranked row from GET /leaderboard.
type LeaderboardEntry struct {
	Username     string  `json:"username"`
	AverageScore float64 `json:"average_score"`
	CommitCount  int     `json:"commit_count"`
}

// LeaderboardResult is the full GET /leaderboard response: the ranked rows
// plus the ranking rules the backend applied (so the CLI never hardcodes
// them).
type LeaderboardResult struct {
	Entries    []LeaderboardEntry `json:"leaderboard"`
	WindowDays int                `json:"window_days"`
	MinCommits int                `json:"min_commits"`
}

// Leaderboard fetches the public leaderboard (no auth required).
func (c *Client) Leaderboard(ctx context.Context) (*LeaderboardResult, error) {
	var out LeaderboardResult
	if err := c.getJSON(ctx, "/leaderboard", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

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

func (c *Client) postJSON(ctx context.Context, path string, body, out any) error {
	return c.postAuthJSON(ctx, path, "", body, out)
}

func (c *Client) postAuthJSON(ctx context.Context, path, token string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

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
