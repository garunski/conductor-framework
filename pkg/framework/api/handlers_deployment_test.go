package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

func TestUp_NoReconciler(t *testing.T) {
	handler, err := newTestHandler(t, WithNilReconciler())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("POST", "/api/up", nil)
	w := httptest.NewRecorder()

	handler.Up(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Up() status code = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Up() error response is not valid JSON: %v", err)
	}

	if errResp.Error != "reconciler_unavailable" {
		t.Errorf("Up() error = %v, want %v", errResp.Error, "reconciler_unavailable")
	}
}

func TestUp_AllServices(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add test manifests
	testManifest1 := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest1)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/up", nil)
	w := httptest.NewRecorder()

	handler.Up(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Up() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Up() response is not valid JSON: %v", err)
	}

	if !strings.Contains(resp["message"], "all services") {
		t.Errorf("Up() message = %v, want to contain 'all services'", resp["message"])
	}
}

func TestUp_WithServices(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add test manifests
	testManifest1 := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest1)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	reqBody := `{"services": ["test-service"]}`
	req := httptest.NewRequest("POST", "/api/up", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Up(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Up() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Up() response is not valid JSON: %v", err)
	}

	if !strings.Contains(resp["message"], "test-service") {
		t.Errorf("Up() message = %v, want to contain 'test-service'", resp["message"])
	}
}

func TestUp_InvalidJSON(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("POST", "/api/up", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Up(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Up() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestUp_NoManifestsForServices(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	reqBody := `{"services": ["nonexistent-service"]}`
	req := httptest.NewRequest("POST", "/api/up", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Up(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Up() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Up() error response is not valid JSON: %v", err)
	}

	if errResp.Error != "no_manifests" {
		t.Errorf("Up() error = %v, want %v", errResp.Error, "no_manifests")
	}
}

func TestDown_NoReconciler(t *testing.T) {
	handler, err := newTestHandler(t, WithNilReconciler())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("POST", "/api/down", nil)
	w := httptest.NewRecorder()

	handler.Down(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Down() status code = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}
}

func TestDown_AllServices(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("POST", "/api/down", nil)
	w := httptest.NewRecorder()

	handler.Down(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Down() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Down() response is not valid JSON: %v", err)
	}

	if !strings.Contains(resp["message"], "all services") {
		t.Errorf("Down() message = %v, want to contain 'all services'", resp["message"])
	}
}

func TestDown_WithServices(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add test manifests
	testManifest1 := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest1)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	reqBody := `{"services": ["test-service"]}`
	req := httptest.NewRequest("POST", "/api/down", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Down(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Down() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Down() response is not valid JSON: %v", err)
	}

	if !strings.Contains(resp["message"], "test-service") {
		t.Errorf("Down() message = %v, want to contain 'test-service'", resp["message"])
	}
}

func TestUpdate_NoReconciler(t *testing.T) {
	handler, err := newTestHandler(t, WithNilReconciler())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("POST", "/api/update", nil)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Update() status code = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}
}

func TestUpdate_AllServices(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add test manifests
	testManifest1 := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest1)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/update", nil)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Update() response is not valid JSON: %v", err)
	}

	if !strings.Contains(resp["message"], "all services") {
		t.Errorf("Update() message = %v, want to contain 'all services'", resp["message"])
	}
}

func TestUpdate_WithServices(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add test manifests
	testManifest1 := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest1)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	reqBody := `{"services": ["test-service"]}`
	req := httptest.NewRequest("POST", "/api/update", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Update() response is not valid JSON: %v", err)
	}

	if !strings.Contains(resp["message"], "test-service") {
		t.Errorf("Update() message = %v, want to contain 'test-service'", resp["message"])
	}
}

func TestFilterManifestsByServices(t *testing.T) {
	tests := []struct {
		name     string
		manifests map[string][]byte
		services  []string
		wantKeys  []string
	}{
		{
			name: "empty services returns all",
			manifests: map[string][]byte{
				"default/Service/test": []byte("test"),
			},
			services: []string{},
			wantKeys: []string{"default/Service/test"},
		},
		{
			name: "filters by service name",
			manifests: map[string][]byte{
				"default/Service/test-service": []byte("test1"),
				"default/Service/other-service": []byte("test2"),
			},
			services: []string{"test-service"},
			wantKeys: []string{"default/Service/test-service"},
		},
		{
			name: "matches service with suffix",
			manifests: map[string][]byte{
				"default/Service/test-service-backend": []byte("test1"),
				"default/Service/test-service-pvc": []byte("test2"),
			},
			services: []string{"test-service"},
			wantKeys: []string{"default/Service/test-service-backend", "default/Service/test-service-pvc"},
		},
		{
			name: "matches service with prefix",
			manifests: map[string][]byte{
				"default/Service/test-service-api": []byte("test1"),
			},
			services: []string{"test-service"},
			wantKeys: []string{"default/Service/test-service-api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterManifestsByServices(tt.manifests, tt.services)
			
			if len(got) != len(tt.wantKeys) {
				t.Errorf("filterManifestsByServices() returned %d manifests, want %d", len(got), len(tt.wantKeys))
			}

			for _, key := range tt.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("filterManifestsByServices() missing key %s", key)
				}
			}
		})
	}
}

func TestUpdateManifestsWithCurrentParameters(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	ctx := context.Background()

	// Create a CRD spec with namespace
	spec := map[string]interface{}{
		"global": map[string]interface{}{
			"namespace": "test-namespace",
		},
	}

	err = handler.parameterClient.CreateWithSpec(ctx, crd.DefaultName, "default", spec)
	if err != nil {
		t.Fatalf("failed to create CRD spec: %v", err)
	}

	// Create manifest in default namespace
	testManifest := createTestManifest("Service", "test-service", "default")
	manifests := map[string][]byte{
		"default/Service/test-service": []byte(testManifest),
	}

	updated, err := handler.updateManifestsWithCurrentParameters(ctx, manifests, crd.DefaultName)
	if err != nil {
		t.Fatalf("updateManifestsWithCurrentParameters() error = %v", err)
	}

	// Check that namespace was updated
	if updatedManifest, ok := updated["test-namespace/Service/test-service"]; ok {
		if !strings.Contains(string(updatedManifest), "namespace: test-namespace") {
			t.Error("updateManifestsWithCurrentParameters() did not update namespace in manifest")
		}
	} else {
		t.Error("updateManifestsWithCurrentParameters() did not create updated manifest with new namespace")
	}
}

func TestUpdateManifestsWithCurrentParameters_NoSpec(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	ctx := context.Background()

	testManifest := createTestManifest("Service", "test-service", "default")
	manifests := map[string][]byte{
		"default/Service/test-service": []byte(testManifest),
	}

	updated, err := handler.updateManifestsWithCurrentParameters(ctx, manifests, crd.DefaultName)
	if err != nil {
		t.Fatalf("updateManifestsWithCurrentParameters() error = %v", err)
	}

	// Should return original manifests when spec doesn't exist
	if len(updated) != len(manifests) {
		t.Errorf("updateManifestsWithCurrentParameters() returned %d manifests, want %d", len(updated), len(manifests))
	}
}

