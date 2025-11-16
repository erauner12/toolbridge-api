package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erauner12/toolbridge-api/internal/auth"
	"github.com/erauner12/toolbridge-api/internal/service/syncservice"
	"github.com/google/uuid"
)

const testUserID = "test-user"

// TestGetNote_IncludeDeleted tests the includeDeleted query parameter behavior
func TestGetNote_IncludeDeleted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := getTestDB(t)
	defer pool.Close()

	srv := &Server{
		DB:              pool,
		RateLimitConfig: DefaultRateLimitConfig,
		NoteSvc:         syncservice.NewNoteService(pool),
	}
	router := srv.Routes(auth.JWTCfg{HS256Secret: "test-secret", DevMode: true})

	ctx := context.Background()
	userID := testUserID

	t.Run("404_for_nonexistent_note", func(t *testing.T) {
		nonexistentUID := uuid.New()

		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/notes/%s", nonexistentUID), nil)
		req.Header.Set("X-Debug-Sub", testUserID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("410_Gone_for_deleted_note_without_includeDeleted", func(t *testing.T) {
		// Create a note via REST API
		noteUID := uuid.New()
		notePayload := map[string]any{
			"uid":     noteUID.String(),
			"title":   "Test Note",
			"content": "This will be deleted",
		}

		_, err := srv.NoteSvc.ApplyNoteMutation(ctx, userID, notePayload, syncservice.MutationOpts{})
		if err != nil {
			t.Fatalf("Failed to create note: %v", err)
		}

		// Soft-delete the note
		_, err = srv.NoteSvc.ApplyNoteMutation(ctx, userID, map[string]any{
			"uid": noteUID.String(),
		}, syncservice.MutationOpts{SetDeleted: true})
		if err != nil {
			t.Fatalf("Failed to delete note: %v", err)
		}

		// Try to GET without includeDeleted flag (should get 410)
		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/notes/%s", noteUID), nil)
		req.Header.Set("X-Debug-Sub", testUserID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusGone {
			t.Errorf("Expected 410 Gone, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp["error"] != "note deleted" {
			t.Errorf("Expected error='note deleted', got %v", resp["error"])
		}

		if resp["deletedAt"] == nil {
			t.Error("Expected deletedAt timestamp in response")
		}
	})

	t.Run("200_OK_for_deleted_note_with_includeDeleted_true", func(t *testing.T) {
		// Create another note
		noteUID := uuid.New()
		notePayload := map[string]any{
			"uid":     noteUID.String(),
			"title":   "Test Note 2",
			"content": "This will be deleted but retrieved",
		}

		_, err := srv.NoteSvc.ApplyNoteMutation(ctx, userID, notePayload, syncservice.MutationOpts{})
		if err != nil {
			t.Fatalf("Failed to create note: %v", err)
		}

		// Soft-delete the note
		_, err = srv.NoteSvc.ApplyNoteMutation(ctx, userID, map[string]any{
			"uid": noteUID.String(),
		}, syncservice.MutationOpts{SetDeleted: true})
		if err != nil {
			t.Fatalf("Failed to delete note: %v", err)
		}

		// GET with includeDeleted=true (should get 200 with full item)
		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/notes/%s?includeDeleted=true", noteUID), nil)
		req.Header.Set("X-Debug-Sub", testUserID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp syncservice.RESTItem
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.UID != noteUID.String() {
			t.Errorf("Expected UID %s, got %s", noteUID, resp.UID)
		}

		if resp.DeletedAt == nil {
			t.Error("Expected deletedAt field to be populated")
		}

		if resp.Payload == nil {
			t.Error("Expected payload to be present")
		}
	})

	t.Run("200_OK_for_active_note", func(t *testing.T) {
		// Create an active note
		noteUID := uuid.New()
		notePayload := map[string]any{
			"uid":     noteUID.String(),
			"title":   "Active Note",
			"content": "This is not deleted",
		}

		_, err := srv.NoteSvc.ApplyNoteMutation(ctx, userID, notePayload, syncservice.MutationOpts{})
		if err != nil {
			t.Fatalf("Failed to create note: %v", err)
		}

		// GET active note (should always return 200)
		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/notes/%s", noteUID), nil)
		req.Header.Set("X-Debug-Sub", testUserID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp syncservice.RESTItem
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.UID != noteUID.String() {
			t.Errorf("Expected UID %s, got %s", noteUID, resp.UID)
		}

		if resp.DeletedAt != nil {
			t.Error("Expected deletedAt to be nil for active note")
		}
	})
}

// TestGetTask_IncludeDeleted tests the same behavior for tasks
func TestGetTask_IncludeDeleted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := getTestDB(t)
	defer pool.Close()

	// Clean tasks table
	_, err := pool.Exec(context.Background(), "DELETE FROM task")
	if err != nil {
		t.Fatalf("Failed to clean tasks table: %v", err)
	}

	srv := &Server{
		DB:              pool,
		RateLimitConfig: DefaultRateLimitConfig,
		TaskSvc:         syncservice.NewTaskService(pool),
	}
	router := srv.Routes(auth.JWTCfg{HS256Secret: "test-secret", DevMode: true})

	ctx := context.Background()
	userID := testUserID

	t.Run("410_Gone_for_deleted_task", func(t *testing.T) {
		taskUID := uuid.New()
		taskPayload := map[string]any{
			"uid":         taskUID.String(),
			"title":       "Test Task",
			"description": "This will be deleted",
		}

		_, err := srv.TaskSvc.ApplyTaskMutation(ctx, userID, taskPayload, syncservice.MutationOpts{})
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Delete the task
		_, err = srv.TaskSvc.ApplyTaskMutation(ctx, userID, map[string]any{
			"uid": taskUID.String(),
		}, syncservice.MutationOpts{SetDeleted: true})
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		// GET without includeDeleted
		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/tasks/%s", taskUID), nil)
		req.Header.Set("X-Debug-Sub", testUserID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusGone {
			t.Errorf("Expected 410 Gone, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("200_OK_for_deleted_task_with_includeDeleted", func(t *testing.T) {
		taskUID := uuid.New()
		taskPayload := map[string]any{
			"uid":   taskUID.String(),
			"title": "Test Task 2",
		}

		_, err := srv.TaskSvc.ApplyTaskMutation(ctx, userID, taskPayload, syncservice.MutationOpts{})
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Delete the task
		_, err = srv.TaskSvc.ApplyTaskMutation(ctx, userID, map[string]any{
			"uid": taskUID.String(),
		}, syncservice.MutationOpts{SetDeleted: true})
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		// GET with includeDeleted=true
		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/tasks/%s?includeDeleted=true", taskUID), nil)
		req.Header.Set("X-Debug-Sub", testUserID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp syncservice.RESTItem
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.DeletedAt == nil {
			t.Error("Expected deletedAt to be populated")
		}
	})
}

// TestMutationOnTombstone tests that mutation handlers return 410 for deleted items
func TestMutationOnTombstone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := getTestDB(t)
	defer pool.Close()

	srv := &Server{
		DB:              pool,
		RateLimitConfig: DefaultRateLimitConfig,
		NoteSvc:         syncservice.NewNoteService(pool),
	}
	router := srv.Routes(auth.JWTCfg{HS256Secret: "test-secret", DevMode: true})

	ctx := context.Background()
	userID := testUserID

	// Create and delete a note
	noteUID := uuid.New()
	_, err := srv.NoteSvc.ApplyNoteMutation(ctx, userID, map[string]any{
		"uid":     noteUID.String(),
		"title":   "Test",
		"content": "Original",
	}, syncservice.MutationOpts{})
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	_, err = srv.NoteSvc.ApplyNoteMutation(ctx, userID, map[string]any{
		"uid": noteUID.String(),
	}, syncservice.MutationOpts{SetDeleted: true})
	if err != nil {
		t.Fatalf("Failed to delete note: %v", err)
	}

	// Try to PATCH the tombstone (should get 410)
	t.Run("PATCH_returns_410_for_tombstone", func(t *testing.T) {
		patchPayload := map[string]any{"content": "Updated"}
		body := toJSONReader(patchPayload)

		req := httptest.NewRequest("PATCH", fmt.Sprintf("/v1/notes/%s", noteUID), body)
		req.Header.Set("X-Debug-Sub", testUserID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusGone {
			t.Errorf("Expected 410 Gone for PATCH on tombstone, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Try to DELETE the tombstone again (should get 410)
	t.Run("DELETE_returns_410_for_tombstone", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/notes/%s", noteUID), nil)
		req.Header.Set("X-Debug-Sub", testUserID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusGone {
			t.Errorf("Expected 410 Gone for DELETE on tombstone, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// toJSONReader converts a payload to a JSON reader for HTTP request bodies
func toJSONReader(payload interface{}) *bytes.Reader {
	data, _ := json.Marshal(payload)
	return bytes.NewReader(data)
}
