package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/garunski/conductor-framework/pkg/framework/database"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"github.com/garunski/conductor-framework/pkg/framework/index"
	"github.com/garunski/conductor-framework/pkg/framework/reconciler"
	"github.com/garunski/conductor-framework/pkg/framework/store"
)

func setupTestReconciler(t *testing.T, ready bool) *reconciler.Reconciler {
	// Create a test reconciler inline since NewTestReconciler was removed
	logger := logr.Discard()
	clientset := kubefake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-store-db")
	testDB, err := database.NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	manifestStore := store.NewManifestStore(testDB, idx, logger)
	eventStore := events.NewStorage(testDB, logger)

	rec, err := reconciler.NewReconciler(clientset, dynamicClient, manifestStore, logger, eventStore)
	if err != nil {
		t.Fatalf("failed to create reconciler: %v", err)
	}
	rec.SetReady(ready)
	return rec
}

func setupTestHandler(t *testing.T) (*Handler, *database.DB) {
	handler, err := NewTestHandler(t)
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
	}

	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("NewTestDB() error = %v", err)
	}
	return handler, db
}

func setupTestHandlerWithReconciler(t *testing.T, rec *reconciler.Reconciler) (*Handler, *database.DB) {
	handler, err := NewTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
	}
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("NewTestDB() error = %v", err)
	}
	return handler, db
}

func setupTestHandlerWithEventStore(t *testing.T) (*Handler, *database.DB, *events.Storage) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("NewTestDB() error = %v", err)
	}
	eventStore := events.NewStorage(db, logr.Discard())
	handler, err := NewTestHandler(t, WithTestEventStore(eventStore))
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
	}
	return handler, db, eventStore
}

func createTestManifest(kind, name, namespace string) string {
	ns := ""
	if namespace != "" {
		ns = "  namespace: " + namespace + "\n"
	}
	return `apiVersion: v1
kind: ` + kind + `
metadata:
  name: ` + name + `
` + ns + `spec: {}
`
}

func TestHealthz(t *testing.T) {
	handler, err := NewTestHandler(t)
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
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
	handler, err := NewTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
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
	handler, err := NewTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
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

func TestGetManifest_NotFound(t *testing.T) {
	handler, err := NewTestHandler(t)
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
	}

	router := handler.SetupRoutes()
	req := httptest.NewRequest("GET", "/manifests/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetManifest() status code = %v, want %v", w.Code, http.StatusNotFound)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("GetManifest() error response is not valid JSON: %v", err)
	}

	if errResp.Error != "not_found" {
		t.Errorf("GetManifest() error = %v, want %v", errResp.Error, "not_found")
	}
}

func TestServiceDetails_InvalidNamespace(t *testing.T) {
	handler, err := NewTestHandler(t)
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
	}

	router := handler.SetupRoutes()
	req := httptest.NewRequest("GET", "/api/service/INVALID_NAMESPACE/service", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServiceDetails() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("ServiceDetails() error response is not valid JSON: %v", err)
	}

	if errResp.Error != "invalid_namespace" {
		t.Errorf("ServiceDetails() error = %v, want %v", errResp.Error, "invalid_namespace")
	}
}

func TestCORSHeaders(t *testing.T) {
	handler, err := NewTestHandler(t)
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
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
	handler, err := NewTestHandler(t)
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
	}

	router := handler.SetupRoutes()

	req := httptest.NewRequest("OPTIONS", "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS request status code = %v, want %v", w.Code, http.StatusNoContent)
	}
}

func TestReadyz_DatabaseUnhealthy(t *testing.T) {

	t.Skip("Cannot detect closed database with current store implementation")
}

func TestReadyz_EventStoreUnavailable(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := NewTestHandler(t, WithTestReconciler(rec), WithNilEventStore())
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
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
	handler, err := NewTestHandler(t, WithNilReconciler())
	if err != nil {
		t.Fatalf("NewTestHandler() error = %v", err)
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
