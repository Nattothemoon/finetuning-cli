package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const DefaultBaseURL = "https://pub.finetuning.ai"

// Version is overridden at build time via -ldflags.
var Version = "dev"

// Client talks to the public finetuning.ai API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTP       *http.Client
	UserAgent  string
	VerboseLog func(format string, args ...any)
}

// NewClient builds a client. baseURL may be empty to use the default.
func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		APIKey:    apiKey,
		HTTP:      &http.Client{Timeout: 60 * time.Second},
		UserAgent: fmt.Sprintf("finetuning-cli/%s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH),
	}
}

// ValidateKey returns true if the key has the expected ft_live_ prefix and length.
func ValidateKey(key string) bool {
	return strings.HasPrefix(key, "ft_live_") && len(key) == 40
}

// envelope is the standard {"data": ...} success wrapper.
type envelope[T any] struct {
	Data T `json:"data"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return err
	}
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	if c.VerboseLog != nil {
		c.VerboseLog("→ %s %s (key=%s)", method, req.URL.Path, redactKey(c.APIKey))
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if c.VerboseLog != nil {
		c.VerboseLog("← %d %s (%d bytes)", resp.StatusCode, resp.Status, len(respBody))
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp.StatusCode, respBody, resp.Header.Get("Retry-After"))
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// Health hits /health (no auth) — useful for `ft doctor`.
func (c *Client) Health(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	prevKey := c.APIKey
	c.APIKey = ""
	defer func() { c.APIKey = prevKey }()
	if err := c.do(ctx, http.MethodGet, "/health", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// IsAPIError extracts an *APIError from a returned error, if present.
func IsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

func redactKey(k string) string {
	if k == "" {
		return "<none>"
	}
	if len(k) <= 12 {
		return "****"
	}
	return k[:12] + "****"
}
