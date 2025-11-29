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

	"github.com/garunski/conductor-framework/pkg/framework/crd"
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

	rec, err := reconciler.NewReconciler(clientset, dynamicClient, manifestStore, logger, eventStore, "test-app")
	if err != nil {
		t.Fatalf("failed to create reconciler: %v", err)
	}
	rec.SetReady(ready)
	return rec
}

func newTestHandler(t *testing.T, opts ...testHandlerOption) (*Handler, error) {
	t.Helper()
	logger := logr.Discard()
	
	// Default options
	cfg := testHandlerConfig{
		appName:    "test-app",
		version:    "test-version",
		logger:     logger,
		reconcileCh: make(chan string, 100),
	}
	
	// Apply options
	for _, opt := range opts {
		opt(&cfg)
	}
	
	// Create database if not provided
	if cfg.db == nil {
		db, err := database.NewTestDB(t)
		if err != nil {
			return nil, err
		}
		cfg.db = db
	}
	
	// Create index and store if not provided
	if cfg.store == nil {
		idx := index.NewIndex()
		cfg.store = store.NewManifestStore(cfg.db, idx, logger)
	}
	
	// Create event store if not provided and not explicitly set to nil
	if cfg.eventStore == nil && !cfg.eventStoreSet {
		cfg.eventStore = events.NewStorage(cfg.db, logger)
	}
	
	// Create parameter client if not provided
	if cfg.parameterClient == nil {
		scheme := runtime.NewScheme()
		corev1.AddToScheme(scheme)
		appsv1.AddToScheme(scheme)
		dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
		cfg.parameterClient = crd.NewClient(dynamicClient, logger, "conductor.io", "v1alpha1", "deploymentparameters")
	}
	
	return NewHandler(cfg.store, cfg.eventStore, cfg.logger, cfg.reconcileCh, cfg.reconciler, cfg.appName, cfg.version, cfg.parameterClient, nil)
}

type testHandlerConfig struct {
	appName         string
	version         string
	logger          logr.Logger
	reconcileCh     chan string
	reconciler      *reconciler.Reconciler
	db              *database.DB
	store           *store.ManifestStore
	eventStore      *events.Storage
	eventStoreSet   bool // Track if eventStore was explicitly set (even if nil)
	parameterClient *crd.Client
}

type testHandlerOption func(*testHandlerConfig)

func WithTestReconciler(rec *reconciler.Reconciler) testHandlerOption {
	return func(cfg *testHandlerConfig) {
		cfg.reconciler = rec
	}
}

func WithTestEventStore(eventStore *events.Storage) testHandlerOption {
	return func(cfg *testHandlerConfig) {
		cfg.eventStore = eventStore
		cfg.eventStoreSet = true
	}
}

func WithNilReconciler() testHandlerOption {
	return func(cfg *testHandlerConfig) {
		cfg.reconciler = nil
	}
}

func WithNilEventStore() testHandlerOption {
	return func(cfg *testHandlerConfig) {
		cfg.eventStore = nil
		cfg.eventStoreSet = true
	}
}

func setupTestHandler(t *testing.T) (*Handler, *database.DB) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("NewTestDB() error = %v", err)
	}
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}
	return handler, db
}

func setupTestHandlerWithReconciler(t *testing.T, rec *reconciler.Reconciler) (*Handler, *database.DB) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("NewTestDB() error = %v", err)
	}
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}
	return handler, db
}

func setupTestHandlerWithEventStore(t *testing.T) (*Handler, *database.DB, *events.Storage) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("NewTestDB() error = %v", err)
	}
	eventStore := events.NewStorage(db, logr.Discard())
	handler, err := newTestHandler(t, WithTestEventStore(eventStore))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
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

func TestGetManifest_NotFound(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
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
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
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
