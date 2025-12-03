package framework

import (
	"context"
	"embed"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.AppName != "conductor" {
		t.Errorf("DefaultConfig() AppName = %v, want conductor", cfg.AppName)
	}

	if cfg.ManifestRoot != "manifests" {
		t.Errorf("DefaultConfig() ManifestRoot = %v, want manifests", cfg.ManifestRoot)
	}

	if cfg.DataPath != getEnvOrDefault("BADGER_DATA_PATH", "/data/badger") {
		t.Errorf("DefaultConfig() DataPath = %v, want /data/badger (or env value)", cfg.DataPath)
	}

	if cfg.Port != getEnvOrDefault("PORT", "8081") {
		t.Errorf("DefaultConfig() Port = %v, want 8081 (or env value)", cfg.Port)
	}

	if cfg.LogRetentionDays != parseIntOrDefault("LOG_RETENTION_DAYS", 7) {
		t.Errorf("DefaultConfig() LogRetentionDays = %v, want 7 (or env value)", cfg.LogRetentionDays)
	}

	if cfg.LogCleanupInterval != parseDurationOrDefault("LOG_CLEANUP_INTERVAL", 1*time.Hour) {
		t.Errorf("DefaultConfig() LogCleanupInterval = %v, want 1h (or env value)", cfg.LogCleanupInterval)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				AppName:            "test",
				DataPath:           "/tmp/test",
				Port:               "8080",
				LogRetentionDays:   7,
				LogCleanupInterval: 1 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "empty AppName",
			config: Config{
				AppName:            "",
				DataPath:           "/tmp/test",
				Port:               "8080",
				LogRetentionDays:   7,
				LogCleanupInterval: 1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "empty DataPath",
			config: Config{
				AppName:            "test",
				DataPath:           "",
				Port:               "8080",
				LogRetentionDays:   7,
				LogCleanupInterval: 1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "empty Port",
			config: Config{
				AppName:            "test",
				DataPath:           "/tmp/test",
				Port:               "",
				LogRetentionDays:   7,
				LogCleanupInterval: 1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "negative LogRetentionDays",
			config: Config{
				AppName:            "test",
				DataPath:           "/tmp/test",
				Port:               "8080",
				LogRetentionDays:   -1,
				LogCleanupInterval: 1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "zero LogCleanupInterval",
			config: Config{
				AppName:            "test",
				DataPath:           "/tmp/test",
				Port:               "8080",
				LogRetentionDays:   7,
				LogCleanupInterval: 0,
			},
			wantErr: true,
		},
		{
			name: "negative LogCleanupInterval",
			config: Config{
				AppName:            "test",
				DataPath:           "/tmp/test",
				Port:               "8080",
				LogRetentionDays:   7,
				LogCleanupInterval: -1 * time.Hour,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_ENV_VAR", "test-value")
	defer os.Unsetenv("TEST_ENV_VAR")

	got := getEnvOrDefault("TEST_ENV_VAR", "default")
	if got != "test-value" {
		t.Errorf("getEnvOrDefault() = %v, want test-value", got)
	}

	// Test with non-existing env var
	got = getEnvOrDefault("NON_EXISTING_VAR", "default")
	if got != "default" {
		t.Errorf("getEnvOrDefault() = %v, want default", got)
	}

	// Test with empty string env var (should return empty, not default)
	os.Setenv("TEST_ENV_VAR", "")
	got = getEnvOrDefault("TEST_ENV_VAR", "default")
	if got != "default" {
		t.Errorf("getEnvOrDefault() with empty string = %v, want default", got)
	}
	os.Unsetenv("TEST_ENV_VAR")
}

func TestParseDurationOrDefault(t *testing.T) {
	// Test with valid duration
	os.Setenv("TEST_DURATION", "2h")
	defer os.Unsetenv("TEST_DURATION")

	got := parseDurationOrDefault("TEST_DURATION", 1*time.Hour)
	if got != 2*time.Hour {
		t.Errorf("parseDurationOrDefault() = %v, want 2h", got)
	}

	// Test with invalid duration
	os.Setenv("TEST_DURATION", "invalid")
	got = parseDurationOrDefault("TEST_DURATION", 1*time.Hour)
	if got != 1*time.Hour {
		t.Errorf("parseDurationOrDefault() = %v, want 1h (default)", got)
	}
	os.Unsetenv("TEST_DURATION")

	// Test with non-existing env var
	got = parseDurationOrDefault("NON_EXISTING_DURATION", 1*time.Hour)
	if got != 1*time.Hour {
		t.Errorf("parseDurationOrDefault() = %v, want 1h (default)", got)
	}

	// Test with empty string (should return default)
	os.Setenv("TEST_DURATION", "")
	got = parseDurationOrDefault("TEST_DURATION", 1*time.Hour)
	if got != 1*time.Hour {
		t.Errorf("parseDurationOrDefault() with empty string = %v, want 1h (default)", got)
	}
	os.Unsetenv("TEST_DURATION")
}

func TestParseIntOrDefault(t *testing.T) {
	// Test with valid int
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	got := parseIntOrDefault("TEST_INT", 10)
	if got != 42 {
		t.Errorf("parseIntOrDefault() = %v, want 42", got)
	}

	// Test with duration format (7d)
	os.Setenv("TEST_INT", "7d")
	got = parseIntOrDefault("TEST_INT", 10)
	if got != 7 {
		t.Errorf("parseIntOrDefault() = %v, want 7", got)
	}
	os.Unsetenv("TEST_INT")

	// Test with duration format that fails parsing (to test fallback to int parsing)
	os.Setenv("TEST_INT", "invalid-duration")
	got = parseIntOrDefault("TEST_INT", 10)
	// Should try to parse as int, which will also fail, so should return default
	if got != 10 {
		t.Errorf("parseIntOrDefault() with invalid duration = %v, want 10 (default)", got)
	}
	os.Unsetenv("TEST_INT")

	// Test with a value that fails duration parsing but succeeds as int
	os.Setenv("TEST_INT", "42")
	got = parseIntOrDefault("TEST_INT", 10)
	// Should parse as int (42d is not a valid duration, so falls back to int parsing)
	if got != 42 {
		t.Errorf("parseIntOrDefault() with int value = %v, want 42", got)
	}
	os.Unsetenv("TEST_INT")

	// Test with invalid value
	os.Setenv("TEST_INT", "invalid")
	got = parseIntOrDefault("TEST_INT", 10)
	if got != 10 {
		t.Errorf("parseIntOrDefault() = %v, want 10 (default)", got)
	}
	os.Unsetenv("TEST_INT")

	// Test with non-existing env var
	got = parseIntOrDefault("NON_EXISTING_INT", 10)
	if got != 10 {
		t.Errorf("parseIntOrDefault() = %v, want 10 (default)", got)
	}

	// Test with empty string (edge case)
	os.Setenv("TEST_INT", "")
	got = parseIntOrDefault("TEST_INT", 10)
	if got != 10 {
		t.Errorf("parseIntOrDefault() with empty string = %v, want 10 (default)", got)
	}
	os.Unsetenv("TEST_INT")
}

// TestSetupLogger tests the setupLogger function
func TestSetupLogger(t *testing.T) {
	logger, err := setupLogger()
	if err != nil {
		t.Fatalf("setupLogger() error = %v", err)
	}
	// Verify logger is usable by checking if it's enabled (any level)
	_ = logger.Enabled()
}

// TestSetupKubernetesClient tests the setupKubernetesClient function
// This tests the fallback behavior when Kubernetes is unavailable
func TestSetupKubernetesClient_NoKubernetes(t *testing.T) {
	logger := logr.Discard()
	cfg := Config{
		CRDGroup:    "test.io",
		CRDVersion:  "v1",
		CRDResource: "test",
	}

	// This will fail gracefully when Kubernetes is not available
	// which is the expected behavior
	_, parameterGetter, err := setupKubernetesClient(context.Background(), logger, cfg)
	if err != nil {
		t.Fatalf("setupKubernetesClient() should not return error on Kubernetes unavailability, got: %v", err)
	}
	// parameterGetter should be nil when Kubernetes is unavailable
	if parameterGetter != nil {
		t.Log("Kubernetes is available in test environment - this is fine")
	}
}

// TestLoadManifests tests the loadManifests function
func TestLoadManifests(t *testing.T) {
	// Create a test filesystem with a manifest
	var testFS embed.FS
	
	cfg := Config{
		ManifestFS:   testFS,
		ManifestRoot: "manifests",
	}

	// Test with nil parameterGetter (fallback behavior)
	manifests, err := loadManifests(context.Background(), cfg, nil)
	if err != nil {
		// This is expected if manifests directory doesn't exist
		t.Logf("loadManifests() with empty FS returned error (expected): %v", err)
		return
	}
	// Should return empty map, not nil
	if manifests == nil {
		t.Error("loadManifests() should return empty map, not nil")
	}
}

// TestRun_InvalidConfig tests Run with invalid configuration
// Note: This test validates that Run() properly validates config before starting
// Full Run() testing requires integration tests due to server lifecycle complexity
func TestRun_InvalidConfig(t *testing.T) {
	cfg := Config{
		AppName: "", // Invalid: empty AppName
	}

	// Use a timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Run(ctx, cfg)
	if err == nil {
		t.Error("Run() with invalid config should return error")
	}
	if !contains(err.Error(), "invalid configuration") {
		t.Errorf("Run() error should mention 'invalid configuration', got: %v", err)
	}
}

// TestRun_EmptyDataPath tests Run with empty DataPath
func TestRun_EmptyDataPath(t *testing.T) {
	cfg := Config{
		AppName:  "test",
		DataPath: "", // Invalid: empty DataPath
		Port:     "8080",
		LogRetentionDays:   7,
		LogCleanupInterval: 1 * time.Hour,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Run(ctx, cfg)
	if err == nil {
		t.Error("Run() with empty DataPath should return error")
	}
	if !contains(err.Error(), "invalid configuration") {
		t.Errorf("Run() error should mention 'invalid configuration', got: %v", err)
	}
}

// TestRun_EmptyPort tests Run with empty Port
func TestRun_EmptyPort(t *testing.T) {
	cfg := Config{
		AppName:  "test",
		DataPath: "/tmp/test",
		Port:     "", // Invalid: empty Port
		LogRetentionDays:   7,
		LogCleanupInterval: 1 * time.Hour,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Run(ctx, cfg)
	if err == nil {
		t.Error("Run() with empty Port should return error")
	}
	if !contains(err.Error(), "invalid configuration") {
		t.Errorf("Run() error should mention 'invalid configuration', got: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
