package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHomePage(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.HomePage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HomePage() status code = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("HomePage() returned empty body")
	}
}

func TestParametersPage(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/parameters", nil)
	w := httptest.NewRecorder()

	handler.ParametersPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ParametersPage() status code = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("ParametersPage() returned empty body")
	}
}

func TestLogsPage(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/logs", nil)
	w := httptest.NewRecorder()

	handler.LogsPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("LogsPage() status code = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("LogsPage() returned empty body")
	}
}

