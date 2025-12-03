package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/garunski/conductor-framework/pkg/framework/events"
)

func TestListEvents_NoEventStore(t *testing.T) {
	handler, err := newTestHandler(t, WithNilEventStore())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/events", nil)
	w := httptest.NewRecorder()

	handler.ListEvents(w, req)

	// WriteError returns 503 for ErrEventStore
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("ListEvents() status code = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}
}

func TestListEvents_Success(t *testing.T) {
	handler, _, eventStore := setupTestHandlerWithEventStore(t)

	// Add a test event
	testEvent := events.Info("test/key", "test", "test event")
	if err := eventStore.StoreEvent(testEvent); err != nil {
		t.Fatalf("failed to store test event: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/events", nil)
	w := httptest.NewRecorder()

	handler.ListEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListEvents() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var eventList []events.Event
	if err := json.Unmarshal(w.Body.Bytes(), &eventList); err != nil {
		t.Fatalf("ListEvents() response is not valid JSON: %v", err)
	}

	if len(eventList) == 0 {
		t.Error("ListEvents() returned no events")
	}
}

func TestGetEventsByResource_NoEventStore(t *testing.T) {
	handler, err := newTestHandler(t, WithNilEventStore())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/events/default/Service/test", nil)
	w := httptest.NewRecorder()

	handler.GetEventsByResource(w, req)

	// WriteError returns 503 for ErrEventStore
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("GetEventsByResource() status code = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}
}

func TestGetEventsByResource_Success(t *testing.T) {
	handler, _, eventStore := setupTestHandlerWithEventStore(t)

	// Add a test event
	testEvent := events.Info("default/Service/test", "test", "test event")
	if err := eventStore.StoreEvent(testEvent); err != nil {
		t.Fatalf("failed to store test event: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/events/default/Service/test?limit=10", nil)
	w := httptest.NewRecorder()

	handler.GetEventsByResource(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetEventsByResource() status code = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestGetEventsByResource_InvalidLimit(t *testing.T) {
	handler, _, _ := setupTestHandlerWithEventStore(t)

	req := httptest.NewRequest("GET", "/api/events/default/Service/test?limit=invalid", nil)
	w := httptest.NewRecorder()

	handler.GetEventsByResource(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GetEventsByResource() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestGetEventsByResource_LimitTooHigh(t *testing.T) {
	handler, _, _ := setupTestHandlerWithEventStore(t)

	req := httptest.NewRequest("GET", "/api/events/default/Service/test?limit=2000", nil)
	w := httptest.NewRecorder()

	handler.GetEventsByResource(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GetEventsByResource() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestGetRecentErrors_NoEventStore(t *testing.T) {
	handler, err := newTestHandler(t, WithNilEventStore())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/events/errors", nil)
	w := httptest.NewRecorder()

	handler.GetRecentErrors(w, req)

	// WriteError returns 503 for ErrEventStore
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("GetRecentErrors() status code = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}
}

func TestGetRecentErrors_Success(t *testing.T) {
	handler, _, eventStore := setupTestHandlerWithEventStore(t)

	// Add a test error event
	testEvent := events.Error("test/key", "test", "test error", errors.New("test error"))
	if err := eventStore.StoreEvent(testEvent); err != nil {
		t.Fatalf("failed to store test event: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/events/errors?limit=10", nil)
	w := httptest.NewRecorder()

	handler.GetRecentErrors(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetRecentErrors() status code = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestCleanupEvents_NoEventStore(t *testing.T) {
	handler, err := newTestHandler(t, WithNilEventStore())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("DELETE", "/api/events?before=2023-01-01T00:00:00Z", nil)
	w := httptest.NewRecorder()

	handler.CleanupEvents(w, req)

	// WriteError returns 503 for ErrEventStore
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("CleanupEvents() status code = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}
}

func TestCleanupEvents_MissingBefore(t *testing.T) {
	handler, _, _ := setupTestHandlerWithEventStore(t)

	req := httptest.NewRequest("DELETE", "/api/events", nil)
	w := httptest.NewRecorder()

	handler.CleanupEvents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CleanupEvents() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestCleanupEvents_InvalidBefore(t *testing.T) {
	handler, _, _ := setupTestHandlerWithEventStore(t)

	req := httptest.NewRequest("DELETE", "/api/events?before=invalid", nil)
	w := httptest.NewRecorder()

	handler.CleanupEvents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CleanupEvents() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestCleanupEvents_Success(t *testing.T) {
	handler, _, _ := setupTestHandlerWithEventStore(t)

	before := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	req := httptest.NewRequest("DELETE", "/api/events?before="+before, nil)
	w := httptest.NewRecorder()

	handler.CleanupEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CleanupEvents() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CleanupEvents() response is not valid JSON: %v", err)
	}

	if resp["message"] != "Events cleaned up successfully" {
		t.Errorf("CleanupEvents() message = %v, want %v", resp["message"], "Events cleaned up successfully")
	}
}

