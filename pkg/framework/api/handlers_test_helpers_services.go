package api

import (
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

// setupTestReconciler creates a test reconciler for services tests
func setupTestReconciler(t *testing.T, ready bool) reconciler.Reconciler {
	t.Helper()
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

// setupTestHandlerWithReconciler creates a test handler with a reconciler for services tests
func setupTestHandlerWithReconciler(t *testing.T, rec reconciler.Reconciler) (*Handler, *database.DB) {
	t.Helper()
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

