package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// CreateSoundEffectRequest carries the body for POST /v1/sound-effects.
// Only Prompt is required. Enhance is sent unconditionally (see the note on
// CreateInstrumentalRequest): the server only treats an explicit `false` as
// "verbatim", so a default-true flag must serialize even when false.
type CreateSoundEffectRequest struct {
	Prompt   string  `json:"prompt"`
	Duration float64 `json:"duration,omitempty"`
	Enhance  bool    `json:"enhance"`
	Webhook  string  `json:"webhook,omitempty"`
}

// SoundEffect is the API representation of a sound effect (create, list, and
// detail share this shape). SFX live in their own store, separate from music
// generations, and each batch flattens to a single clip's audioUrl/duration.
type SoundEffect struct {
	ID                string           `json:"id"`
	Status            GenerationStatus `json:"status"`
	Prompt            string           `json:"prompt"`
	RequestedDuration float64          `json:"requestedDuration"`
	AudioURL          string           `json:"audioUrl"`
	Duration          float64          `json:"duration"`
	Webhook           *string          `json:"webhook,omitempty"`
	DailyRemaining    int              `json:"dailyRemaining,omitempty"`
	ErrorMessage      string           `json:"errorMessage,omitempty"`
	CreatedAt         string           `json:"createdAt"`
	CompletedAt       string           `json:"completedAt"`
}

// SfxListResult wraps the paginated sound-effects list response.
type SfxListResult struct {
	SoundEffects []SoundEffect `json:"soundEffects"`
	HasMore      bool          `json:"hasMore"`
	NextOffset   int           `json:"nextOffset"`
}

// CreateSoundEffect → POST /v1/sound-effects.
func (c *Client) CreateSoundEffect(ctx context.Context, req *CreateSoundEffectRequest) (*SoundEffect, error) {
	var out envelope[SoundEffect]
	if err := c.do(ctx, http.MethodPost, "/v1/sound-effects", req, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// ListSoundEffects → GET /v1/sound-effects.
func (c *Client) ListSoundEffects(ctx context.Context, opts ListOptions) (*SfxListResult, error) {
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
	path := "/v1/sound-effects"
	if encoded := q.Encode(); encoded != "" {
		path = path + "?" + encoded
	}
	var out envelope[SfxListResult]
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// GetSoundEffect → GET /v1/sound-effects/:id.
func (c *Client) GetSoundEffect(ctx context.Context, id string) (*SoundEffect, error) {
	var out envelope[SoundEffect]
	if err := c.do(ctx, http.MethodGet, "/v1/sound-effects/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}
