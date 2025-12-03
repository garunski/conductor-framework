package reconciler

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
	"github.com/garunski/conductor-framework/pkg/framework/store"
)

func setupTestReconcilerForTests(t *testing.T) *Reconciler {
	logger := logr.Discard()
	clientset := kubefake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-reconciler-db")
	testDB, err := database.NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	manifestStore := store.NewManifestStore(testDB, idx, logger)
	eventStore := events.NewStorage(testDB, logger)

	rec, err := NewReconciler(clientset, dynamicClient, manifestStore, logger, eventStore, "test-app")
	if err != nil {
		t.Fatalf("failed to create reconciler: %v", err)
	}
	return rec
}
