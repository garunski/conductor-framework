package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"

	"github.com/garunski/conductor-framework/pkg/framework/manifest"
)

func TestClusterRequirements_NoReconciler(t *testing.T) {
	handler, err := newTestHandler(t, WithNilReconciler())
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/cluster/requirements", nil)
	w := httptest.NewRecorder()

	handler.ClusterRequirements(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("ClusterRequirements() status code = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

func TestClusterRequirements_Success(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/cluster/requirements", nil)
	w := httptest.NewRecorder()

	handler.ClusterRequirements(w, req)

	// Should return OK even if no requirements file exists
	if w.Code != http.StatusOK {
		t.Errorf("ClusterRequirements() status code = %v, want %v", w.Code, http.StatusOK)
	}

	var resp ClusterRequirementsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("ClusterRequirements() response is not valid JSON: %v", err)
	}

	if resp.Overall == "" {
		t.Error("ClusterRequirements() response missing overall status")
	}
}

func TestProcessApplicationRequirement(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	ctx := context.Background()
	clientset := rec.GetClientset()

	// Test unknown check type
	appReq := manifest.ApplicationRequirement{
		Name:        "test",
		Description: "test requirement",
		CheckType:   "unknown-type",
		Required:    true,
	}

	req := handler.processApplicationRequirement(ctx, clientset, appReq, nil, nil, nil)
	if req == nil {
		t.Error("processApplicationRequirement() should return requirement for unknown type")
	}
	if req.Status != "warning" {
		t.Errorf("processApplicationRequirement() Status = %v, want warning", req.Status)
	}
}

func TestCheckKubernetesVersion(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	discovery := rec.GetClientset().Discovery()
	versionInfo, _ := discovery.ServerVersion()

	// Test with valid version
	appReq := manifest.ApplicationRequirement{
		Name:        "k8s-version",
		Description: "Kubernetes version check",
		CheckType:   "kubernetes-version",
		CheckConfig: map[string]interface{}{
			"minimumVersion": "1.0",
		},
		Required: true,
	}

	req := handler.checkKubernetesVersion(appReq, discovery, versionInfo)
	if req == nil {
		t.Error("checkKubernetesVersion() should return requirement")
	}
	if req.Status == "" {
		t.Error("checkKubernetesVersion() should set status")
	}
}

func TestCheckNodeCount(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Test with nil nodes
	appReq := manifest.ApplicationRequirement{
		Name:        "node-count",
		Description: "Node count check",
		CheckType:   "node-count",
		CheckConfig: map[string]interface{}{
			"minimum": 1,
		},
		Required: true,
	}

	req := handler.checkNodeCount(appReq, nil)
	if req == nil {
		t.Error("checkNodeCount() should return requirement")
	}
	if req.Status != "fail" {
		t.Errorf("checkNodeCount() Status = %v, want fail for nil nodes", req.Status)
	}

	// Test with empty node list
	nodes := &corev1.NodeList{Items: []corev1.Node{}}
	req = handler.checkNodeCount(appReq, nodes)
	if req == nil {
		t.Error("checkNodeCount() should return requirement")
	}
}

func TestCheckStorageClass(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	ctx := context.Background()
	clientset := rec.GetClientset()

	// Test with specific storage class name
	appReq := manifest.ApplicationRequirement{
		Name:        "storage-class",
		Description: "Storage class check",
		CheckType:   "storage-class",
		CheckConfig: map[string]interface{}{
			"name": "standard",
		},
		Required: true,
	}

	storageClasses := &storagev1.StorageClassList{Items: []storagev1.StorageClass{}}
	req := handler.checkStorageClass(appReq, clientset, ctx, storageClasses)
	if req == nil {
		t.Error("checkStorageClass() should return requirement")
	}
	if req.Status != "fail" {
		t.Errorf("checkStorageClass() Status = %v, want fail for missing storage class", req.Status)
	}
}

func TestCheckCPU(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Test with nil nodes
	appReq := manifest.ApplicationRequirement{
		Name:        "cpu",
		Description: "CPU check",
		CheckType:   "cpu",
		CheckConfig: map[string]interface{}{
			"minimum": "2",
		},
		Required: true,
	}

	req := handler.checkCPU(appReq, nil)
	if req == nil {
		t.Error("checkCPU() should return requirement")
	}
	if req.Status != "fail" {
		t.Errorf("checkCPU() Status = %v, want fail for nil nodes", req.Status)
	}
}

func TestCheckMemory(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Test with nil nodes
	appReq := manifest.ApplicationRequirement{
		Name:        "memory",
		Description: "Memory check",
		CheckType:   "memory",
		CheckConfig: map[string]interface{}{
			"minimum": "4Gi",
		},
		Required: true,
	}

	req := handler.checkMemory(appReq, nil)
	if req == nil {
		t.Error("checkMemory() should return requirement")
	}
	if req.Status != "fail" {
		t.Errorf("checkMemory() Status = %v, want fail for nil nodes", req.Status)
	}
}
