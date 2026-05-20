package api

import (
	"context"
	"net/http"
)

type Me struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Tier   string `json:"tier"`
	Limits struct {
		MonthlyGenerations int `json:"monthlyGenerations"`
		GenerationsUsed    int `json:"generationsUsed"`
		MonthlyRemaining   int `json:"monthlyRemaining"`
		PackCredits        int `json:"packCredits"`
		TotalRemaining     int `json:"totalRemaining"`
		QueueDepth         int `json:"queueDepth"`
	} `json:"limits"`
}

func (c *Client) Me(ctx context.Context) (*Me, error) {
	var out envelope[Me]
	if err := c.do(ctx, http.MethodGet, "/v1/me", nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}
