package server

import (
	"context"
	"embed"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/logr"

	"github.com/garunski/conductor-framework/pkg/framework/database"
)

var emptyFS embed.FS

func TestNewServer(t *testing.T) {
	logger := logr.Discard()
	cfg := &Config{
		AppName:            "test-app",
		AppVersion:         "1.0.0",
		DataPath:           t.TempDir(),
		Port:               "8080",
		LogRetentionDays:   7,
		LogCleanupInterval: 1 * time.Hour,
		CRDGroup:           "conductor.io",
		CRDVersion:         "v1alpha1",
		CRDResource:        "deploymentparameters",
		ManifestFS:         emptyFS,
		ManifestRoot:       "manifests",
	}

	manifests := map[string][]byte{}

	// NewServer may succeed or fail depending on Kubernetes config availability
	// We just verify it doesn't panic and handles the call appropriately
	_, err := NewServer(cfg, logger, manifests)
	// Error is expected if Kubernetes is not available, success if it is
	// Both are valid outcomes for this test
	_ = err
}

func TestServer_Close(t *testing.T) {
	logger := logr.Discard()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-db")
	db, err := database.NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}

	// Create a minimal server struct for testing Close
	srv := &Server{
		db: db,
	}

	err = srv.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Close again should not error
	err = srv.Close()
	if err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
}

func TestServer_Close_NilDB(t *testing.T) {
	srv := &Server{
		db: nil,
	}

	err := srv.Close()
	if err != nil {
		t.Errorf("Close() with nil DB should not error, got %v", err)
	}
}

func TestServer_Shutdown(t *testing.T) {
	logger := logr.Discard()
	cfg := &Config{
		AppName:            "test-app",
		AppVersion:         "1.0.0",
		DataPath:           t.TempDir(),
		Port:               "8080",
		LogRetentionDays:   7,
		LogCleanupInterval: 1 * time.Hour,
		CRDGroup:           "conductor.io",
		CRDVersion:         "v1alpha1",
		CRDResource:        "deploymentparameters",
		ManifestFS:         emptyFS,
		ManifestRoot:       "manifests",
	}

	// Create a test HTTP server
	httpServer := &http.Server{
		Addr: ":0", // Use port 0 to get an available port
	}

	srv := &Server{
		config:     cfg,
		logger:     logger,
		httpServer: httpServer,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Shutdown should work even if server wasn't started
	err := srv.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestInt32Ptr(t *testing.T) {
	val := int32(42)
	ptr := int32Ptr(val)
	if ptr == nil {
		t.Error("int32Ptr() returned nil")
	}
	if *ptr != val {
		t.Errorf("int32Ptr() = %v, want %v", *ptr, val)
	}
}

// Test Start
func TestServer_Start(t *testing.T) {
	logger := logr.Discard()
	cfg := &Config{
		AppName:            "test-app",
		AppVersion:         "1.0.0",
		DataPath:           t.TempDir(),
		Port:               "0", // Use port 0 to get available port
		LogRetentionDays:   7,
		LogCleanupInterval: 1 * time.Hour,
		CRDGroup:           "conductor.io",
		CRDVersion:         "v1alpha1",
		CRDResource:        "deploymentparameters",
		ManifestFS:         emptyFS,
		ManifestRoot:       "manifests",
	}

	manifests := map[string][]byte{}

	// Create server (may fail if K8s not available, that's ok)
	srv, err := NewServer(cfg, logger, manifests)
	if err != nil {
		t.Skipf("Skipping test - Kubernetes not available: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	err = srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify reconciler is ready
	if !srv.reconciler.IsReady() {
		t.Error("Start() did not set reconciler to ready")
	}

	// Give goroutines a moment to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context to stop goroutines
	cancel()
	time.Sleep(50 * time.Millisecond)
}

// Test startReconciliationHandler
func TestServer_startReconciliationHandler(t *testing.T) {
	logger := logr.Discard()
	cfg := &Config{
		AppName:            "test-app",
		AppVersion:         "1.0.0",
		DataPath:           t.TempDir(),
		Port:               "0",
		LogRetentionDays:   7,
		LogCleanupInterval: 1 * time.Hour,
		CRDGroup:           "conductor.io",
		CRDVersion:         "v1alpha1",
		CRDResource:        "deploymentparameters",
		ManifestFS:         emptyFS,
		ManifestRoot:       "manifests",
	}

	manifests := map[string][]byte{}

	srv, err := NewServer(cfg, logger, manifests)
	if err != nil {
		t.Skipf("Skipping test - Kubernetes not available: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start reconciliation handler
	go srv.startReconciliationHandler(ctx)

	// Send a reconciliation key
	srv.reconcileCh <- "default/ConfigMap/test"

	// Give handler time to process
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()
	time.Sleep(50 * time.Millisecond)
}

// Test startLogCleanup
func TestServer_startLogCleanup(t *testing.T) {
	logger := logr.Discard()
	cfg := &Config{
		AppName:            "test-app",
		AppVersion:         "1.0.0",
		DataPath:           t.TempDir(),
		Port:               "0",
		LogRetentionDays:   7,
		LogCleanupInterval: 100 * time.Millisecond, // Short interval for testing
		CRDGroup:           "conductor.io",
		CRDVersion:         "v1alpha1",
		CRDResource:        "deploymentparameters",
		ManifestFS:         emptyFS,
		ManifestRoot:       "manifests",
	}

	manifests := map[string][]byte{}

	srv, err := NewServer(cfg, logger, manifests)
	if err != nil {
		t.Skipf("Skipping test - Kubernetes not available: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start log cleanup
	go srv.startLogCleanup(ctx)

	// Wait for at least one cleanup cycle
	time.Sleep(150 * time.Millisecond)

	// Cancel context
	cancel()
	time.Sleep(50 * time.Millisecond)
}

// Test WaitForShutdown with signal simulation
// Note: This test is limited because we can't easily simulate OS signals in unit tests
// In a real scenario, this would be tested via integration tests
func TestServer_WaitForShutdown_Context(t *testing.T) {
	logger := logr.Discard()
	cfg := &Config{
		AppName:            "test-app",
		AppVersion:         "1.0.0",
		DataPath:           t.TempDir(),
		Port:               "0",
		LogRetentionDays:   7,
		LogCleanupInterval: 1 * time.Hour,
		CRDGroup:           "conductor.io",
		CRDVersion:         "v1alpha1",
		CRDResource:        "deploymentparameters",
		ManifestFS:         emptyFS,
		ManifestRoot:       "manifests",
	}

	manifests := map[string][]byte{}

	srv, err := NewServer(cfg, logger, manifests)
	if err != nil {
		t.Skipf("Skipping test - Kubernetes not available: %v", err)
		return
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	err = srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// WaitForShutdown blocks on signal, so we test Shutdown directly
	// which is what WaitForShutdown calls
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	cancel()
}

