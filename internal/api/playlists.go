package api

import (
	"context"
	"net/http"
	"net/url"
)

// BulkLimit is the server-side cap on ids per bulk request. Callers with more
// ids must chunk and merge results.
const BulkLimit = 100

// Playlist is the API representation of a user playlist (GET /v1/playlists).
type Playlist struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"` // may be null → ""
	TrackCount    int    `json:"trackCount"`
	TotalDuration int    `json:"totalDuration"` // seconds
	IsPublic      bool   `json:"isPublic"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// ItemError is a per-item failure inside a bulk response. bulk-delete reports
// the failing id under "id"; the playlist endpoints use "trackId".
type ItemError struct {
	ID      string `json:"id,omitempty"`
	TrackID string `json:"trackId,omitempty"`
	Error   string `json:"error"`
}

// Key returns whichever id field the endpoint populated.
func (e ItemError) Key() string {
	if e.TrackID != "" {
		return e.TrackID
	}
	return e.ID
}

// BulkDeleteResult is the response of POST /v1/generations/bulk-delete.
type BulkDeleteResult struct {
	Deleted []string    `json:"deleted"`
	Errors  []ItemError `json:"errors,omitempty"`
}

// AddTracksResult is the response of POST /v1/playlists/:id/tracks.
type AddTracksResult struct {
	Added  []string    `json:"added"`
	Errors []ItemError `json:"errors,omitempty"`
}

// RemoveTracksResult is the response of POST /v1/playlists/:id/tracks/bulk-remove.
type RemoveTracksResult struct {
	Removed []string    `json:"removed"`
	Errors  []ItemError `json:"errors,omitempty"`
}

// MoveTracksResult is the response of POST /v1/playlists/:id/tracks/move.
type MoveTracksResult struct {
	Moved  []string    `json:"moved"`
	Errors []ItemError `json:"errors,omitempty"`
}

// PlaylistDetail is the response of GET /v1/playlists/:id — a playlist plus
// every track in it. Works on own playlists and other users' public ones.
type PlaylistDetail struct {
	Playlist
	IsOwner bool         `json:"isOwner"`
	Tracks  []Generation `json:"tracks"`
}

// GetPlaylist → GET /v1/playlists/:id.
func (c *Client) GetPlaylist(ctx context.Context, id string) (*PlaylistDetail, error) {
	var out envelope[PlaylistDetail]
	if err := c.do(ctx, http.MethodGet, "/v1/playlists/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// ListPlaylists → GET /v1/playlists. Returns all of the caller's playlists.
func (c *Client) ListPlaylists(ctx context.Context) ([]Playlist, error) {
	var out envelope[struct {
		Playlists []Playlist `json:"playlists"`
	}]
	if err := c.do(ctx, http.MethodGet, "/v1/playlists", nil, &out); err != nil {
		return nil, err
	}
	return out.Data.Playlists, nil
}

// BulkDeleteGenerations → POST /v1/generations/bulk-delete. Max BulkLimit ids.
// Always 200 with partial results; per-id failures land in Errors.
func (c *Client) BulkDeleteGenerations(ctx context.Context, ids []string) (*BulkDeleteResult, error) {
	body := map[string][]string{"ids": ids}
	var out envelope[BulkDeleteResult]
	if err := c.do(ctx, http.MethodPost, "/v1/generations/bulk-delete", body, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// AddPlaylistTracks → POST /v1/playlists/:id/tracks. Max BulkLimit ids.
// When zero tracks could be added the server replies 400 ADD_FAILED; that is
// folded into a normal result (empty Added, populated Errors) so callers
// handle partial and total failure the same way.
func (c *Client) AddPlaylistTracks(ctx context.Context, playlistID string, trackIDs []string) (*AddTracksResult, error) {
	body := map[string][]string{"trackIds": trackIDs}
	var out envelope[AddTracksResult]
	err := c.do(ctx, http.MethodPost, "/v1/playlists/"+url.PathEscape(playlistID)+"/tracks", body, &out)
	if err != nil {
		if items, ok := allFailedItems(err, "ADD_FAILED"); ok {
			return &AddTracksResult{Errors: items}, nil
		}
		return nil, err
	}
	return &out.Data, nil
}

// RemovePlaylistTracks → POST /v1/playlists/:id/tracks/bulk-remove. Max
// BulkLimit ids. Always 200, even when nothing was removed.
func (c *Client) RemovePlaylistTracks(ctx context.Context, playlistID string, trackIDs []string) (*RemoveTracksResult, error) {
	body := map[string][]string{"trackIds": trackIDs}
	var out envelope[RemoveTracksResult]
	if err := c.do(ctx, http.MethodPost, "/v1/playlists/"+url.PathEscape(playlistID)+"/tracks/bulk-remove", body, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// MovePlaylistTracks → POST /v1/playlists/:id/tracks/move. Max BulkLimit ids.
// A track that fails to add to the target stays in the source. The all-failed
// 400 MOVE_FAILED is folded into a normal result, like AddPlaylistTracks.
func (c *Client) MovePlaylistTracks(ctx context.Context, sourceID, targetID string, trackIDs []string) (*MoveTracksResult, error) {
	body := map[string]any{"trackIds": trackIDs, "targetPlaylistId": targetID}
	var out envelope[MoveTracksResult]
	err := c.do(ctx, http.MethodPost, "/v1/playlists/"+url.PathEscape(sourceID)+"/tracks/move", body, &out)
	if err != nil {
		if items, ok := allFailedItems(err, "MOVE_FAILED"); ok {
			return &MoveTracksResult{Errors: items}, nil
		}
		return nil, err
	}
	return &out.Data, nil
}

// allFailedItems unwraps the per-item errors from an all-failed bulk response.
func allFailedItems(err error, code string) ([]ItemError, bool) {
	apiErr, ok := IsAPIError(err)
	if !ok || apiErr.Code != code {
		return nil, false
	}
	return apiErr.DetailItems, true
}
