package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// APIError is the structured error returned by pub.finetuning.ai.
type APIError struct {
	HTTPStatus int            `json:"-"`
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	// DetailItems is populated instead of Details when the server sends
	// `details` as an array of per-item errors (ADD_FAILED / MOVE_FAILED).
	DetailItems []ItemError    `json:"-"`
	RetryAfter  int            `json:"retryAfter,omitempty"`
	Raw         map[string]any `json:"-"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

// FieldDetail returns the human-readable per-field validation message, when present.
func (e *APIError) FieldDetail() string {
	if e == nil || e.Details == nil {
		return ""
	}
	if f, ok := e.Details["field"].(string); ok && f != "" {
		return f
	}
	return ""
}

// parseAPIError reads a non-2xx response body into APIError. Always returns a non-nil error.
func parseAPIError(status int, body []byte, retryAfterHeader string) error {
	var envelope struct {
		Error struct {
			Code       string          `json:"code"`
			Message    string          `json:"message"`
			Details    json.RawMessage `json:"details"`
			RetryAfter int             `json:"retryAfter"`
		} `json:"error"`
	}
	err := &APIError{HTTPStatus: status}
	if jsonErr := json.Unmarshal(body, &envelope); jsonErr == nil && envelope.Error.Code != "" {
		err.Code = envelope.Error.Code
		err.Message = envelope.Error.Message
		err.RetryAfter = envelope.Error.RetryAfter
		// `details` is an object on most routes, but an array of per-item
		// errors on the bulk all-failed responses. Try both shapes.
		if len(envelope.Error.Details) > 0 {
			if json.Unmarshal(envelope.Error.Details, &err.Details) != nil {
				_ = json.Unmarshal(envelope.Error.Details, &err.DetailItems)
			}
		}
	} else {
		err.Code = fmt.Sprintf("HTTP_%d", status)
		err.Message = string(body)
		if err.Message == "" {
			err.Message = fmt.Sprintf("unexpected HTTP %d", status)
		}
	}
	if err.RetryAfter == 0 && retryAfterHeader != "" {
		if v, parseErr := strconv.Atoi(retryAfterHeader); parseErr == nil {
			err.RetryAfter = v
		}
	}
	return err
}
