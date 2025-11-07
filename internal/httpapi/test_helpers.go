package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// TestSession holds session info for testing
type TestSession struct {
	ID    string
	Epoch int
}

// createTestSession creates a sync session for testing and returns session info
func createTestSession(t *testing.T, router http.Handler) TestSession {
	t.Helper()

	req := httptest.NewRequest("POST", "/v1/sync/sessions", nil)
	req.Header.Set("X-Debug-Sub", "test-user")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("Failed to create session: got status %d, body: %s", w.Code, w.Body.String())
	}

	var session struct {
		ID    string `json:"id"`
		Epoch int    `json:"epoch"`
	}
	if err := json.NewDecoder(w.Body).Decode(&session); err != nil {
		t.Fatalf("Failed to decode session response: %v", err)
	}

	return TestSession{
		ID:    session.ID,
		Epoch: session.Epoch,
	}
}

// makeRequestWithSession makes an HTTP request with X-Sync-Session and X-Sync-Epoch headers
func makeRequestWithSession(t *testing.T, router http.Handler, method, path string, body interface{}, sessionOrID interface{}) *httptest.ResponseRecorder {
	t.Helper()

	// Support both TestSession and string (backwards compatibility)
	var session TestSession
	switch v := sessionOrID.(type) {
	case TestSession:
		session = v
	case string:
		session = TestSession{ID: v, Epoch: 1} // Default epoch for old tests
	default:
		t.Fatalf("sessionOrID must be TestSession or string, got %T", sessionOrID)
	}

	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Debug-Sub", "test-user")
	req.Header.Set("X-Sync-Session", session.ID)
	req.Header.Set("X-Sync-Epoch", strconv.Itoa(session.Epoch))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}
