package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type GenerationStatus string

const (
	StatusPending    GenerationStatus = "pending"
	StatusProcessing GenerationStatus = "processing"
	StatusCompleted  GenerationStatus = "completed"
	StatusFailed     GenerationStatus = "failed"
)

// CreateGenerationRequest carries the body for POST /v1/generations.
// Only Tags is required; zero-valued fields are omitted so the server applies defaults.
type CreateGenerationRequest struct {
	Tags          string `json:"tags"`
	Lyrics        string `json:"lyrics,omitempty"`
	Duration      int    `json:"duration,omitempty"`
	BPM           int    `json:"bpm,omitempty"`
	Key           string `json:"key,omitempty"`
	Scale         string `json:"scale,omitempty"`
	TimeSignature string `json:"timesignature,omitempty"`
	Language      string `json:"language,omitempty"`
	Seed          int64  `json:"seed,omitempty"`
	Webhook       string `json:"webhook,omitempty"`
}

// Generation is the API representation of a track (list + detail share most fields).
type Generation struct {
	ID               string           `json:"id"`
	Title            string           `json:"title"`
	Prompt           string           `json:"prompt"`
	Status           GenerationStatus `json:"status"`
	AudioURL         string           `json:"audioUrl"`
	Duration         int              `json:"duration"`
	IsPublic         bool             `json:"isPublic"`
	PlayCount        int              `json:"playCount"`
	LikeCount        int              `json:"likeCount"`
	Parameters       map[string]any   `json:"parameters"`
	CreatedAt        string           `json:"createdAt"`
	CompletedAt      string           `json:"completedAt"`
	FileSize         int64            `json:"fileSize,omitempty"`
	DownloadCount    int              `json:"downloadCount,omitempty"`
	ErrorMessage     string           `json:"errorMessage,omitempty"`
	GenerationTime   float64          `json:"generationTime,omitempty"`
	CreditsRemaining int              `json:"creditsRemaining,omitempty"`
	Webhook          *string          `json:"webhook,omitempty"`
}

// ListResult wraps the paginated list response.
type ListResult struct {
	Generations []Generation `json:"generations"`
	HasMore     bool         `json:"hasMore"`
	NextOffset  int          `json:"nextOffset"`
}

// ListOptions for GET /v1/generations.
type ListOptions struct {
	Limit  int
	Offset int
	Status GenerationStatus
}

// CreateGeneration → POST /v1/generations.
func (c *Client) CreateGeneration(ctx context.Context, req *CreateGenerationRequest) (*Generation, error) {
	var out envelope[Generation]
	if err := c.do(ctx, http.MethodPost, "/v1/generations", req, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// ListGenerations → GET /v1/generations.
func (c *Client) ListGenerations(ctx context.Context, opts ListOptions) (*ListResult, error) {
	q := url.Values{}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	if opts.Status != "" {
		q.Set("status", string(opts.Status))
	}
	path := "/v1/generations"
	if encoded := q.Encode(); encoded != "" {
		path = path + "?" + encoded
	}
	var out envelope[ListResult]
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// GetGeneration → GET /v1/generations/:id.
func (c *Client) GetGeneration(ctx context.Context, id string) (*Generation, error) {
	var out envelope[Generation]
	if err := c.do(ctx, http.MethodGet, "/v1/generations/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// DownloadAudio streams audioUrl to w. No auth header — the R2 URL is public.
// Returns total bytes written and the Content-Length if the server reported it (0 if unknown).
func (c *Client) DownloadAudio(ctx context.Context, audioURL string, w io.Writer, progress func(written, total int64)) (int64, error) {
	if audioURL == "" {
		return 0, fmt.Errorf("audio url is empty (generation may still be processing)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	// Use a longer timeout for downloads.
	dlClient := &http.Client{Timeout: 5 * time.Minute}
	resp, err := dlClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength
	pr := &progressReader{r: resp.Body, total: total, cb: progress}
	n, err := io.Copy(w, pr)
	return n, err
}
