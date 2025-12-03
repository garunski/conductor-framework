package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestListManifests(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add test manifests
	testManifest1 := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest1)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	req := httptest.NewRequest("GET", "/manifests", nil)
	w := httptest.NewRecorder()

	handler.ListManifests(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListManifests() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var manifests map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &manifests); err != nil {
		t.Fatalf("ListManifests() response is not valid JSON: %v", err)
	}

	if len(manifests) == 0 {
		t.Error("ListManifests() returned no manifests")
	}
}

func TestCreateManifest_Success(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	reqBody := `{
		"key": "default/Service/test-service",
		"value": "apiVersion: v1\nkind: Service\nmetadata:\n  name: test-service\n  namespace: default\nspec: {}"
	}`
	req := httptest.NewRequest("POST", "/manifests", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateManifest(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("CreateManifest() status code = %v, want %v", w.Code, http.StatusCreated)
	}

	// Verify manifest was created
	manifest, ok := handler.store.Get("default/Service/test-service")
	if !ok {
		t.Error("CreateManifest() did not create manifest")
	}

	if !strings.Contains(string(manifest), "test-service") {
		t.Error("CreateManifest() created manifest with wrong content")
	}
}

func TestCreateManifest_InvalidJSON(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("POST", "/manifests", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateManifest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateManifest() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestCreateManifest_InvalidKey(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Use an empty key which will fail ValidateKey
	reqBody := `{
		"key": "",
		"value": "apiVersion: v1\nkind: Service\nmetadata:\n  name: test\nspec: {}"
	}`
	req := httptest.NewRequest("POST", "/manifests", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateManifest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateManifest() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestCreateManifest_InvalidYAML(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	reqBody := `{
		"key": "default/Service/test-service",
		"value": "invalid: yaml: content: ["
	}`
	req := httptest.NewRequest("POST", "/manifests", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateManifest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateManifest() status code = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateManifest_Success(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Create initial manifest
	testManifest := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	reqBody := `{
		"value": "apiVersion: v1\nkind: Service\nmetadata:\n  name: test-service-updated\n  namespace: default\nspec: {}"
	}`
	req := httptest.NewRequest("PUT", "/manifests/default/Service/test-service", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateManifest(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UpdateManifest() status code = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify manifest was updated
	manifest, ok := handler.store.Get("default/Service/test-service")
	if !ok {
		t.Error("UpdateManifest() did not update manifest")
	}

	if !strings.Contains(string(manifest), "test-service-updated") {
		t.Error("UpdateManifest() did not update manifest content")
	}
}

func TestUpdateManifest_NotFound(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	reqBody := `{
		"value": "apiVersion: v1\nkind: Service\nmetadata:\n  name: test\nspec: {}"
	}`
	req := httptest.NewRequest("PUT", "/manifests/default/Service/nonexistent", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateManifest(w, req)

	// UpdateManifest returns 404 when manifest not found
	if w.Code != http.StatusNotFound {
		t.Errorf("UpdateManifest() status code = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestDeleteManifest_Success(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Create initial manifest
	testManifest := createTestManifest("Service", "test-service", "default")
	if err := handler.store.Create("default/Service/test-service", []byte(testManifest)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/manifests/default/Service/test-service", nil)
	w := httptest.NewRecorder()

	handler.DeleteManifest(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("DeleteManifest() status code = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Verify manifest was deleted
	_, ok := handler.store.Get("default/Service/test-service")
	if ok {
		t.Error("DeleteManifest() did not delete manifest")
	}
}

func TestDeleteManifest_NotFound(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("DELETE", "/manifests/default/Service/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.DeleteManifest(w, req)

	// DeleteManifest returns 404 when manifest not found
	if w.Code != http.StatusNotFound {
		t.Errorf("DeleteManifest() status code = %v, want %v", w.Code, http.StatusNotFound)
	}
}

