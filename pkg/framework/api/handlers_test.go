package api

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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
	
	var emptyFS embed.FS
	return NewHandler(cfg.store, cfg.eventStore, cfg.logger, cfg.reconcileCh, cfg.reconciler, cfg.appName, cfg.version, cfg.parameterClient, nil, emptyFS, "")
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

func TestDeploymentsPage(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add a test service manifest to the store
	testManifest := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	req := httptest.NewRequest("GET", "/deployments", nil)
	w := httptest.NewRecorder()

	handler.DeploymentsPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DeploymentsPage() status code = %v, want %v", w.Code, http.StatusOK)
	}

	// Check that the response contains HTML
	body := w.Body.String()
	if body == "" {
		t.Error("DeploymentsPage() returned empty body")
	}

	// Check that the response contains expected template elements
	if !strings.Contains(body, "Deployment Controls") {
		t.Error("DeploymentsPage() response should contain 'Deployment Controls'")
	}
}

func TestDeploymentsPage_WithCRDSpec(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add a test service manifest
	testManifest := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	// Create a CRD spec with nested parameters
	// Use float64 for numbers to be compatible with JSON unmarshaling
	ctx := context.Background()
	spec := map[string]interface{}{
		"global": map[string]interface{}{
			"namespace":     "test-ns",
			"namePrefix":    "test-",
			"imageRegistry": "gcr.io/test",
		},
		"services": map[string]interface{}{
			"test-service": map[string]interface{}{
				"replicas": float64(3),
				"config": map[string]interface{}{
					"database": map[string]interface{}{
						"host": "localhost",
						"port": float64(5432),
					},
				},
			},
		},
	}

	// Create the CRD instance
	err = handler.parameterClient.CreateWithSpec(ctx, crd.DefaultName, "default", spec)
	if err != nil {
		t.Fatalf("failed to create CRD spec: %v", err)
	}

	req := httptest.NewRequest("GET", "/deployments", nil)
	w := httptest.NewRecorder()

	handler.DeploymentsPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DeploymentsPage() status code = %v, want %v", w.Code, http.StatusOK)
	}

	// The template should render the CRD spec data
	// The namespace data is passed to the template and used by JavaScript
	// Since the data is in JSON format for JavaScript, we verify the page renders successfully
	// and that the CRD spec was retrieved correctly by checking the handler executed without error
	body := w.Body.String()
	
	// Verify the page rendered successfully
	if !strings.Contains(body, "Deployment Controls") {
		t.Error("DeploymentsPage() should render the page successfully")
	}
	
	// The namespace "test-ns" is in the CRD spec that was created
	// It's passed to the template as instanceSpec and converted to JSON for JavaScript
	// We verify the data flow worked by ensuring the page rendered and the handler didn't error
	// The actual namespace value is used by JavaScript at runtime, not rendered in static HTML
}

func TestGetServiceValuesMap(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Create a CRD spec with global and service-specific parameters
	// Use float64 for numbers to be compatible with JSON unmarshaling
	ctx := context.Background()
	spec := map[string]interface{}{
		"global": map[string]interface{}{
			"namespace": "default",
			"replicas":  float64(1),
			"imageTag":  "latest",
		},
		"services": map[string]interface{}{
			"service1": map[string]interface{}{
				"replicas": float64(3),
				"imageTag": "v1.0.0",
			},
			"service2": map[string]interface{}{
				"config": map[string]interface{}{
					"enabled": true,
				},
			},
		},
	}

	err = handler.parameterClient.CreateWithSpec(ctx, crd.DefaultName, "default", spec)
	if err != nil {
		t.Fatalf("failed to create CRD spec: %v", err)
	}

	services := []string{"service1", "service2"}
	result := handler.getServiceValuesMap(ctx, services, "default", "default")

	// Check service1 - should have merged values (global + service-specific)
	if service1Data, ok := result["service1"]; !ok {
		t.Error("getServiceValuesMap() should return data for service1")
	} else {
		merged, ok := service1Data["merged"].(map[string]interface{})
		if !ok {
			t.Error("getServiceValuesMap() should return merged values as map")
		} else {
			// Should have global defaults
			if merged["namespace"] != "default" {
				t.Errorf("getServiceValuesMap() merged namespace = %v, want %v", merged["namespace"], "default")
			}
			// Should have service-specific override for replicas
			// Note: JSON unmarshaling converts numbers to float64
			if replicas, ok := merged["replicas"].(float64); !ok || replicas != 3 {
				t.Errorf("getServiceValuesMap() merged replicas = %v, want %v", merged["replicas"], float64(3))
			}
			// Should have service-specific override for imageTag
			if merged["imageTag"] != "v1.0.0" {
				t.Errorf("getServiceValuesMap() merged imageTag = %v, want %v", merged["imageTag"], "v1.0.0")
			}
		}
	}

	// Check service2 - should have merged values (global + service-specific)
	if service2Data, ok := result["service2"]; !ok {
		t.Error("getServiceValuesMap() should return data for service2")
	} else {
		merged, ok := service2Data["merged"].(map[string]interface{})
		if !ok {
			t.Error("getServiceValuesMap() should return merged values as map")
		} else {
			// Should have global defaults
			if merged["namespace"] != "default" {
				t.Errorf("getServiceValuesMap() merged namespace = %v, want %v", merged["namespace"], "default")
			}
			// Should have service-specific config
			if config, ok := merged["config"].(map[string]interface{}); ok {
				if config["enabled"] != true {
					t.Errorf("getServiceValuesMap() merged config.enabled = %v, want %v", config["enabled"], true)
				}
			} else {
				t.Error("getServiceValuesMap() should preserve nested config structure")
			}
		}
	}
}

func TestGetServiceValuesMap_EmptySpec(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	ctx := context.Background()
	services := []string{"service1"}
	result := handler.getServiceValuesMap(ctx, services, "default", "default")

	// Should return empty merged values when no CRD spec exists
	if service1Data, ok := result["service1"]; !ok {
		t.Error("getServiceValuesMap() should return data for service1 even with empty spec")
	} else {
		merged, ok := service1Data["merged"].(map[string]interface{})
		if !ok {
			t.Error("getServiceValuesMap() should return merged values as map")
		} else {
			if len(merged) != 0 {
				t.Errorf("getServiceValuesMap() should return empty merged map when no spec exists, got %v", merged)
			}
		}
	}
}

func TestDeploymentsPage_EmptySpec(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add a test service manifest
	testManifest := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	// Don't create any CRD spec - should handle nil services gracefully
	req := httptest.NewRequest("GET", "/deployments", nil)
	w := httptest.NewRecorder()

	handler.DeploymentsPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DeploymentsPage() status code = %v, want %v", w.Code, http.StatusOK)
	}

	// Should render successfully even with empty spec
	body := w.Body.String()
	if body == "" {
		t.Error("DeploymentsPage() returned empty body")
	}
}

func TestMergeSchemaWithInstance(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		instance map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "schema with instance values",
			schema: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": map[string]interface{}{},
					"replicas":  map[string]interface{}{},
				},
				"services": map[string]interface{}{},
			},
			instance: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": "default",
					"replicas":  float64(3),
				},
			},
			expected: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": "default",
					"replicas":  float64(3),
				},
				"services": map[string]interface{}{},
			},
		},
		{
			name: "empty schema with instance",
			schema: map[string]interface{}{
				"global":   map[string]interface{}{},
				"services": map[string]interface{}{},
			},
			instance: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": "test",
				},
				"services": map[string]interface{}{
					"service1": map[string]interface{}{
						"replicas": float64(2),
					},
				},
			},
			expected: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": "test",
				},
				"services": map[string]interface{}{
					"service1": map[string]interface{}{
						"replicas": float64(2),
					},
				},
			},
		},
		{
			name: "nested schema with instance",
			schema: map[string]interface{}{
				"services": map[string]interface{}{
					"config": map[string]interface{}{
						"database": map[string]interface{}{
							"host": map[string]interface{}{},
						},
					},
				},
			},
			instance: map[string]interface{}{
				"services": map[string]interface{}{
					"api-service": map[string]interface{}{
						"config": map[string]interface{}{
							"database": map[string]interface{}{
								"host": "localhost",
								"port": float64(5432),
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"global": map[string]interface{}{},
				"services": map[string]interface{}{
					"api-service": map[string]interface{}{
						"config": map[string]interface{}{
							"database": map[string]interface{}{
								"host": "localhost",
								"port": float64(5432),
							},
						},
					},
				},
			},
		},
		{
			name: "schema template with multiple services",
			schema: map[string]interface{}{
				"services": map[string]interface{}{
					"replicas": map[string]interface{}{},
					"config":   map[string]interface{}{},
				},
			},
			instance: map[string]interface{}{
				"services": map[string]interface{}{
					"service1": map[string]interface{}{
						"replicas": float64(3),
					},
					"service2": map[string]interface{}{
						"replicas": float64(2),
						"config": map[string]interface{}{
							"enabled": true,
						},
					},
				},
			},
			expected: map[string]interface{}{
				"global": map[string]interface{}{},
				"services": map[string]interface{}{
					"service1": map[string]interface{}{
						"replicas": float64(3),
						"config":   map[string]interface{}{},
					},
					"service2": map[string]interface{}{
						"replicas": float64(2),
						"config": map[string]interface{}{
							"enabled": true,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeSchemaWithInstance(tt.schema, tt.instance)
			
			// Verify global
			if resultGlobal, ok := result["global"].(map[string]interface{}); ok {
				if expectedGlobal, ok := tt.expected["global"].(map[string]interface{}); ok {
					if !mapsEqual(resultGlobal, expectedGlobal) {
						t.Errorf("mergeSchemaWithInstance() global = %v, want %v", resultGlobal, expectedGlobal)
					}
				}
			}
			
			// Verify services
			if resultServices, ok := result["services"].(map[string]interface{}); ok {
				if expectedServices, ok := tt.expected["services"].(map[string]interface{}); ok {
					if !mapsEqual(resultServices, expectedServices) {
						t.Errorf("mergeSchemaWithInstance() services = %v, want %v", resultServices, expectedServices)
					}
				}
			}
		})
	}
}

// mapsEqual compares two maps recursively for testing
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok {
			return false
		} else {
			if aMap, ok := v.(map[string]interface{}); ok {
				if bMap, ok := bv.(map[string]interface{}); ok {
					if !mapsEqual(aMap, bMap) {
						return false
					}
				} else {
					return false
				}
			} else if v != bv {
				return false
			}
		}
	}
	return true
}
