package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)













// mapsEqual compares two maps recursively for testing


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

func TestServiceDetails_ValidNamespace(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	router := handler.SetupRoutes()
	req := httptest.NewRequest("GET", "/api/service/default/test-service", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// ServiceDetails may return 404 if service doesn't exist in cluster
	// Just verify it doesn't return 400 (invalid namespace)
	if w.Code == http.StatusBadRequest {
		t.Errorf("ServiceDetails() status code = %v, should not be BadRequest for valid namespace", w.Code)
	}
}

func TestListServices(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add test service manifest
	testManifest := `apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: default
spec:
  ports:
  - port: 8080`
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/services", nil)
	w := httptest.NewRecorder()

	handler.ListServices(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListServices() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp ServiceListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("ListServices() response is not valid JSON: %v", err)
	}

	if len(resp.Services) == 0 {
		t.Error("ListServices() returned no services")
	}
}

func TestStatus(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/services/health", nil)
	w := httptest.NewRecorder()

	handler.Status(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Status() response is not valid JSON: %v", err)
	}
}

func TestExtractServices(t *testing.T) {
	tests := []struct {
		name      string
		manifests map[string][]byte
		wantCount int
	}{
		{
			name: "single service",
			manifests: map[string][]byte{
				"default/Service/test": []byte(`apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - port: 8080`),
			},
			wantCount: 1,
		},
		{
			name: "no services",
			manifests: map[string][]byte{
				"default/Deployment/test": []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test`),
			},
			wantCount: 0,
		},
		{
			name:      "empty manifests",
			manifests: map[string][]byte{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractServices(context.Background(), tt.manifests)
			if len(got) != tt.wantCount {
				t.Errorf("extractServices() returned %d services, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFindServiceManifests(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add test manifests
	deploymentManifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-service
  namespace: default`
	statefulSetManifest := `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test-service
  namespace: default`

	if err := handler.store.Create("default/Deployment/test-service", []byte(deploymentManifest)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}
	if err := handler.store.Create("default/StatefulSet/test-service", []byte(statefulSetManifest)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	// Should prefer StatefulSet over Deployment
	manifest, err := handler.findServiceManifests("test-service")
	if err != nil {
		t.Fatalf("findServiceManifests() error = %v", err)
	}

	if !strings.Contains(string(manifest), "StatefulSet") {
		t.Error("findServiceManifests() should prefer StatefulSet over Deployment")
	}

	// Test with service that doesn't exist
	_, err = handler.findServiceManifests("nonexistent")
	if err == nil {
		t.Error("findServiceManifests() should return error for nonexistent service")
	}
}

func TestExtractServiceManifest(t *testing.T) {
	manifests := map[string][]byte{
		"default/Service/test": []byte(`apiVersion: v1
kind: Service
metadata:
  name: test
  namespace: default
spec:
  ports:
  - port: 8080`),
	}

	manifest, found := extractServiceManifest(context.Background(), manifests, "default", "test")
	if !found {
		t.Error("extractServiceManifest() returned found=false")
	}
	if manifest == nil {
		t.Error("extractServiceManifest() returned nil")
	}
	if !strings.Contains(string(manifest), "kind: Service") {
		t.Error("extractServiceManifest() did not return correct manifest")
	}

	// Test with non-existent service
	_, found = extractServiceManifest(context.Background(), manifests, "default", "nonexistent")
	if found {
		t.Error("extractServiceManifest() should return found=false for nonexistent service")
	}
}

func TestExtractServiceSelector(t *testing.T) {
	manifest := []byte(`apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  selector:
    app: test-app`)

	selector, err := extractServiceSelector(context.Background(), manifest)
	if err != nil {
		t.Fatalf("extractServiceSelector() error = %v", err)
	}
	if selector == nil {
		t.Error("extractServiceSelector() returned nil")
	}
	if selector["app"] != "test-app" {
		t.Errorf("extractServiceSelector() = %v, want app=test-app", selector)
	}
}

func TestFindMatchingDeployment(t *testing.T) {
	manifests := map[string][]byte{
		"default/Deployment/test": []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
  namespace: default
spec:
  template:
    metadata:
      labels:
        app: test-app`),
	}

	selector := map[string]string{"app": "test-app"}

	// Should find matching deployment
	deployment := findMatchingDeployment(context.Background(), manifests, "default", selector)
	if deployment == nil {
		t.Error("findMatchingDeployment() should return deployment when match found")
	}

	// Should return nil when no match
	deployment = findMatchingDeployment(context.Background(), manifests, "default", map[string]string{"app": "nonexistent"})
	if deployment != nil {
		t.Error("findMatchingDeployment() should return nil when no deployment matches")
	}
}

func TestExtractEnvVars(t *testing.T) {
	deploymentManifest := []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers:
      - name: test
        env:
        - name: ENV1
          value: value1
        - name: ENV2
          value: value2
        - name: ENV3
          valueFrom:
            secretKeyRef:
              name: secret
              key: key`)

	envVars := extractEnvVars(context.Background(), deploymentManifest)
	if len(envVars) != 3 {
		t.Errorf("extractEnvVars() returned %d env vars, want 3", len(envVars))
	}
	if envVars[0].Name != "ENV1" || envVars[0].Value != "value1" {
		t.Error("extractEnvVars() did not extract correct env vars")
	}
	if envVars[2].Source != "secret:secret/key" {
		t.Errorf("extractEnvVars() Source = %v, want secret:secret/key", envVars[2].Source)
	}
}
