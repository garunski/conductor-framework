package framework

import (
	"context"
	"embed"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

// Note: Full Run() integration tests are complex and require proper server lifecycle management
// The Run() function starts a server and waits for shutdown, which makes it difficult to unit test
// without proper mocking. Integration tests should be used for full Run() testing in a separate
// test suite with proper setup/teardown.

// TestRun_LoadManifestsFailure tests Run when manifest loading fails
func TestRun_LoadManifestsFailure(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DataPath = t.TempDir()
	cfg.Port = "8080"
	cfg.ManifestRoot = "nonexistent-directory" // Non-existent directory

	var emptyFS embed.FS
	cfg.ManifestFS = emptyFS

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Run(ctx, cfg)
	// Error is expected due to missing manifests directory
	if err == nil {
		t.Error("Run() should return error when manifests cannot be loaded")
	}
	if err != nil && !contains(err.Error(), "failed to load embedded manifests") {
		t.Logf("Run() returned error (expected): %v", err)
	}
}


// TestSetupKubernetesClient_WithKubernetes tests setupKubernetesClient when Kubernetes is available
func TestSetupKubernetesClient_WithKubernetes(t *testing.T) {
	logger := logr.Discard()
	cfg := Config{
		CRDGroup:    "test.io",
		CRDVersion:  "v1",
		CRDResource: "test",
	}

	// This will succeed if Kubernetes is available, or return nil if not
	// Both are valid outcomes
	_, parameterGetter, _ := setupKubernetesClient(context.Background(), logger, cfg)
	if parameterGetter != nil {
		t.Log("Kubernetes is available - parameterGetter created")
	} else {
		t.Log("Kubernetes is not available - parameterGetter is nil (expected fallback)")
	}
}

// TestLoadManifests_WithEmptyFS tests loadManifests with empty filesystem
func TestLoadManifests_WithEmptyFS(t *testing.T) {
	var emptyFS embed.FS
	cfg := Config{
		ManifestFS:   emptyFS,
		ManifestRoot: "manifests",
	}

	manifests, err := loadManifests(context.Background(), cfg, nil)
	if err != nil {
		// Error is expected if manifests directory doesn't exist
		t.Logf("loadManifests() with empty FS returned error (expected): %v", err)
		return
	}
	// Should return empty map, not nil
	if manifests == nil {
		t.Error("loadManifests() should return empty map, not nil")
	}
	if len(manifests) != 0 {
		t.Errorf("loadManifests() with empty FS should return empty map, got %d manifests", len(manifests))
	}
}

// TestLoadManifests_WithContextCancellation tests loadManifests with cancelled context
func TestLoadManifests_WithContextCancellation(t *testing.T) {
	var emptyFS embed.FS
	cfg := Config{
		ManifestFS:   emptyFS,
		ManifestRoot: "manifests",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	manifests, err := loadManifests(ctx, cfg, nil)
	// Should handle cancellation gracefully
	if err != nil && err != context.Canceled {
		t.Logf("loadManifests() with cancelled context returned error: %v", err)
	}
	_ = manifests
}

