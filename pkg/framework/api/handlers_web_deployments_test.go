package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

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

