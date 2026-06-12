package api

import "testing"

func TestParseAPIErrorObjectDetails(t *testing.T) {
	body := []byte(`{"error":{"code":"VALIDATION_ERROR","message":"too many ids","details":{"max":100}}}`)
	err := parseAPIError(400, body, "")
	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "VALIDATION_ERROR" {
		t.Errorf("code = %q", apiErr.Code)
	}
	if max, _ := apiErr.Details["max"].(float64); max != 100 {
		t.Errorf("details.max = %v", apiErr.Details["max"])
	}
}

func TestParseAPIErrorArrayDetails(t *testing.T) {
	body := []byte(`{"error":{"code":"ADD_FAILED","message":"Cannot add private track to public playlist","details":[{"trackId":"trk_def","error":"Cannot add private track to public playlist"}]}}`)
	err := parseAPIError(400, body, "")
	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "ADD_FAILED" {
		t.Errorf("code = %q", apiErr.Code)
	}
	if len(apiErr.DetailItems) != 1 {
		t.Fatalf("DetailItems = %v", apiErr.DetailItems)
	}
	item := apiErr.DetailItems[0]
	if item.Key() != "trk_def" || item.Error != "Cannot add private track to public playlist" {
		t.Errorf("item = %+v", item)
	}
}
