package config

import (
	"embed"
	"testing"
	"time"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	if builder == nil {
		t.Fatal("NewBuilder() returned nil")
	}
}

func TestBuilder_WithAppName(t *testing.T) {
	builder := NewBuilder()
	result := builder.WithAppName("test-app")
	if result != builder {
		t.Error("WithAppName() should return the same builder")
	}

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.AppName != "test-app" {
		t.Errorf("AppName = %v, want test-app", cfg.AppName)
	}
}

func TestBuilder_WithAppVersion(t *testing.T) {
	builder := NewBuilder()
	builder.WithAppVersion("1.0.0")

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.AppVersion != "1.0.0" {
		t.Errorf("AppVersion = %v, want 1.0.0", cfg.AppVersion)
	}
}

func TestBuilder_WithManifestFS(t *testing.T) {
	var testFS embed.FS
	builder := NewBuilder()
	builder.WithManifestFS(testFS)

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// ManifestFS is an embed.FS which cannot be nil, just check it's set
	_ = cfg.ManifestFS
}

func TestBuilder_WithManifestRoot(t *testing.T) {
	builder := NewBuilder()
	builder.WithManifestRoot("custom/manifests")

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.ManifestRoot != "custom/manifests" {
		t.Errorf("ManifestRoot = %v, want custom/manifests", cfg.ManifestRoot)
	}
}

func TestBuilder_WithCustomTemplateFS(t *testing.T) {
	var testFS embed.FS
	builder := NewBuilder()
	builder.WithCustomTemplateFS(&testFS)

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.CustomTemplateFS == nil {
		t.Error("CustomTemplateFS should not be nil")
	}
}

func TestBuilder_WithDataPath(t *testing.T) {
	builder := NewBuilder()
	builder.WithDataPath("/custom/path")

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.DataPath != "/custom/path" {
		t.Errorf("DataPath = %v, want /custom/path", cfg.DataPath)
	}
}

func TestBuilder_WithPort(t *testing.T) {
	builder := NewBuilder()
	builder.WithPort("9090")

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %v, want 9090", cfg.Port)
	}
}

func TestBuilder_WithLogRetentionDays(t *testing.T) {
	builder := NewBuilder()
	builder.WithLogRetentionDays(14)

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.LogRetentionDays != 14 {
		t.Errorf("LogRetentionDays = %v, want 14", cfg.LogRetentionDays)
	}
}

func TestBuilder_WithLogCleanupInterval(t *testing.T) {
	builder := NewBuilder()
	interval := 2 * time.Hour
	builder.WithLogCleanupInterval(interval)

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.LogCleanupInterval != interval {
		t.Errorf("LogCleanupInterval = %v, want %v", cfg.LogCleanupInterval, interval)
	}
}

func TestBuilder_WithCRDGroup(t *testing.T) {
	builder := NewBuilder()
	builder.WithCRDGroup("custom.io")

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.CRDGroup != "custom.io" {
		t.Errorf("CRDGroup = %v, want custom.io", cfg.CRDGroup)
	}
}

func TestBuilder_WithCRDVersion(t *testing.T) {
	builder := NewBuilder()
	builder.WithCRDVersion("v1beta1")

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.CRDVersion != "v1beta1" {
		t.Errorf("CRDVersion = %v, want v1beta1", cfg.CRDVersion)
	}
}

func TestBuilder_WithCRDResource(t *testing.T) {
	builder := NewBuilder()
	builder.WithCRDResource("customparameters")

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.CRDResource != "customparameters" {
		t.Errorf("CRDResource = %v, want customparameters", cfg.CRDResource)
	}
}

func TestBuilder_Build(t *testing.T) {
	builder := NewBuilder()
	builder.WithAppName("test")
	builder.WithDataPath("/tmp/test")
	builder.WithPort("8080")

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if cfg.AppName != "test" {
		t.Errorf("AppName = %v, want test", cfg.AppName)
	}
}

func TestBuilder_Build_Invalid(t *testing.T) {
	builder := NewBuilder()
	builder.WithAppName("") // Invalid: empty AppName

	_, err := builder.Build()
	if err == nil {
		t.Error("Build() should return error for invalid config")
	}
}

func TestBuilder_MustBuild(t *testing.T) {
	builder := NewBuilder()
	builder.WithAppName("test")
	builder.WithDataPath("/tmp/test")
	builder.WithPort("8080")

	cfg := builder.MustBuild()

	if cfg.AppName != "test" {
		t.Errorf("AppName = %v, want test", cfg.AppName)
	}
}

func TestBuilder_MustBuild_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should panic on invalid config")
		}
	}()

	builder := NewBuilder()
	builder.WithAppName("") // Invalid: empty AppName
	builder.MustBuild()
}

func TestBuilder_Chaining(t *testing.T) {
	cfg := NewBuilder().
		WithAppName("test").
		WithAppVersion("1.0.0").
		WithDataPath("/tmp/test").
		WithPort("8080").
		MustBuild()

	if cfg.AppName != "test" {
		t.Errorf("AppName = %v, want test", cfg.AppName)
	}
	if cfg.AppVersion != "1.0.0" {
		t.Errorf("AppVersion = %v, want 1.0.0", cfg.AppVersion)
	}
}

