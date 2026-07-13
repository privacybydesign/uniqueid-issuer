package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWriteJSONSetsContentType is a regression test for the bug where the
// Content-Type header was set after WriteHeader and therefore silently dropped
// (issue #29). It asserts that the header is present on the response, that the
// status code is passed through, and that the body is written verbatim.
func TestWriteJSONSetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	body := []byte(`{"hello":"world"}`)

	if err := writeJSON(rec, http.StatusOK, body); err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}

	// Inspect the response as a client would see it. Result().Header reflects the
	// header map as it was snapshotted at WriteHeader time, so a Content-Type set
	// *after* WriteHeader (the original bug) would be missing here even though the
	// recorder's live Header() map would still show it.
	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", res.StatusCode, http.StatusOK)
	}

	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	if got := rec.Body.Bytes(); string(got) != string(body) {
		t.Errorf("body = %q, want %q", got, body)
	}

	// The written body must still be valid JSON matching the declared type.
	var v map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Errorf("response body is not valid JSON: %v", err)
	}
}
