package reconciler

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/garunski/conductor-framework/pkg/framework/database"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"github.com/garunski/conductor-framework/pkg/framework/index"
	"github.com/garunski/conductor-framework/pkg/framework/store"
)

func TestNewReconciler(t *testing.T) {
	logger := logr.Discard()
	clientset := kubefake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-db")
	testDB, err := database.NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	manifestStore := store.NewManifestStore(testDB, idx, logger)
	eventStore := events.NewStorage(testDB, logger)

	rec, err := NewReconciler(clientset, dynamicClient, manifestStore, logger, eventStore, "test-app")
	if err != nil {
		t.Fatalf("NewReconciler() error = %v", err)
	}

	if rec == nil {
		t.Fatal("NewReconciler() returned nil")
	}

	if rec.appName != "test-app" {
		t.Errorf("NewReconciler() appName = %v, want test-app", rec.appName)
	}
}

func TestNewReconciler_DefaultAppName(t *testing.T) {
	logger := logr.Discard()
	clientset := kubefake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-db")
	testDB, err := database.NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	manifestStore := store.NewManifestStore(testDB, idx, logger)
	eventStore := events.NewStorage(testDB, logger)

	rec, err := NewReconciler(clientset, dynamicClient, manifestStore, logger, eventStore, "")
	if err != nil {
		t.Fatalf("NewReconciler() error = %v", err)
	}

	if rec.appName != "conductor" {
		t.Errorf("NewReconciler() appName = %v, want conductor", rec.appName)
	}
}

func TestReconciler_SetReady(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	rec.SetReady(true)
	if !rec.IsReady() {
		t.Error("SetReady(true) did not set ready to true")
	}

	rec.SetReady(false)
	if rec.IsReady() {
		t.Error("SetReady(false) did not set ready to false")
	}
}

func TestReconciler_IsReady(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	// Initially should be false
	if rec.IsReady() {
		t.Error("IsReady() should return false initially")
	}

	rec.SetReady(true)
	if !rec.IsReady() {
		t.Error("IsReady() should return true after SetReady(true)")
	}
}

func TestReconciler_GetClientset(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	clientset := rec.GetClientset()
	if clientset == nil {
		t.Error("GetClientset() returned nil")
	}
}

func TestReconciler_WaitForFirstReconciliation(t *testing.T) {
	rec := setupTestReconcilerForTests(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should timeout since no reconciliation has happened
	err := rec.WaitForFirstReconciliation(ctx)
	if err == nil {
		t.Error("WaitForFirstReconciliation() expected timeout error, got nil")
	}
}

func TestReconciler_WaitForFirstReconciliation_ContextCanceled(t *testing.T) {
	rec := setupTestReconcilerForTests(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rec.WaitForFirstReconciliation(ctx)
	if err == nil {
		t.Error("WaitForFirstReconciliation() expected context canceled error, got nil")
	}
}

