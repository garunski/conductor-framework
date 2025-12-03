package framework

import (
	"os"
	"testing"
	"time"
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
}

// Note: Run() is tested indirectly through integration tests
// as it requires a full Kubernetes setup and is complex to unit test
