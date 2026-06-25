package api

import (
	"context"
	"net/http"
)

// CreateInstrumentalRequest carries the body for POST /v1/instrumental.
// Instrumental tracks are vocals-free — the model picks instrumentation and
// tempo from the prompt, so there's no bpm/key/scale/lyrics to send. Only
// Prompt is required; zero-valued fields are omitted so the server defaults.
//
// Enhance is sent unconditionally (no omitempty): the server treats only an
// explicit `false` as "use my words verbatim", so a default-true flag must
// serialize even when false.
type CreateInstrumentalRequest struct {
	Prompt   string `json:"prompt"`
	Duration int    `json:"duration,omitempty"`
	Enhance  bool   `json:"enhance"`
	Seed     int64  `json:"seed,omitempty"`
	Webhook  string `json:"webhook,omitempty"`
}

// CreateInstrumental → POST /v1/instrumental.
//
// The response is the same {data: Generation} envelope as music — instrumental
// rows live in the generations store (type="instrumental"), so reads reuse
// ListGenerations(type=instrumental) and GetGeneration.
func (c *Client) CreateInstrumental(ctx context.Context, req *CreateInstrumentalRequest) (*Generation, error) {
	var out envelope[Generation]
	if err := c.do(ctx, http.MethodPost, "/v1/instrumental", req, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}
