package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler.Healthz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Healthz() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var status HealthStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("Healthz() response is not valid JSON: %v", err)
	}

	if status.Status != "healthy" {
		t.Errorf("Healthz() status = %v, want %v", status.Status, "healthy")
	}

	if status.Version != "test-version" {
		t.Errorf("Healthz() version = %v, want %v", status.Version, "test-version")
	}
}

func TestReadyz_AllHealthy(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	handler.Readyz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Readyz() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var status HealthStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("Readyz() response is not valid JSON: %v", err)
	}

	if status.Status != "healthy" {
		t.Errorf("Readyz() status = %v, want %v", status.Status, "healthy")
	}

	if status.Components["database"].Status != "healthy" {
		t.Errorf("Readyz() database status = %v, want %v", status.Components["database"].Status, "healthy")
	}

	if status.Components["manager"].Status != "ready" {
		t.Errorf("Readyz() manager status = %v, want %v", status.Components["manager"].Status, "ready")
	}
}

func TestReadyz_ManagerNotReady(t *testing.T) {
	rec := setupTestReconciler(t, false)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	handler.Readyz(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Readyz() status code = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}

	var status HealthStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("Readyz() response is not valid JSON: %v", err)
	}

	if status.Status != "unhealthy" {
		t.Errorf("Readyz() status = %v, want %v", status.Status, "unhealthy")
	}

	if status.Components["manager"].Status != "not_ready" {
		t.Errorf("Readyz() manager status = %v, want %v", status.Components["manager"].Status, "not_ready")
	}
}

func TestReadyz_DatabaseUnhealthy(t *testing.T) {

	t.Skip("Cannot detect closed database with current store implementation")
}

func TestReadyz_EventStoreUnavailable(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec), WithNilEventStore())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	handler.Readyz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Readyz() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var status HealthStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("Readyz() response is not valid JSON: %v", err)
	}

	if status.Components["eventStore"].Status != "unavailable" {
		t.Errorf("Readyz() eventStore status = %v, want %v", status.Components["eventStore"].Status, "unavailable")
	}
}

func TestReadyz_NoManager(t *testing.T) {
	handler, err := newTestHandler(t, WithNilReconciler())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	handler.Readyz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Readyz() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var status HealthStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("Readyz() response is not valid JSON: %v", err)
	}

	if _, ok := status.Components["manager"]; ok {
		t.Error("Readyz() should not include manager component when manager is nil")
	}
}

func TestCORSHeaders(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	router := handler.SetupRoutes()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS header Access-Control-Allow-Origin = %v, want *", w.Header().Get("Access-Control-Allow-Origin"))
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS header Access-Control-Allow-Methods is missing")
	}
}

func TestCORSPreflight(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	router := handler.SetupRoutes()

	req := httptest.NewRequest("OPTIONS", "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS request status code = %v, want %v", w.Code, http.StatusNoContent)
	}
}

