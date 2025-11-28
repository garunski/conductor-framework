package config

import (
	"embed"
	"fmt"
	"time"

	"github.com/garunski/conductor-framework/pkg/framework"
)

// Builder provides a fluent interface for building framework configuration.
type Builder struct {
	config framework.Config
}

// NewBuilder creates a new configuration builder with default values.
func NewBuilder() *Builder {
	return &Builder{
		config: framework.DefaultConfig(),
	}
}

// WithAppName sets the application name.
func (b *Builder) WithAppName(name string) *Builder {
	b.config.AppName = name
	return b
}

// WithAppVersion sets the application version.
func (b *Builder) WithAppVersion(version string) *Builder {
	b.config.AppVersion = version
	return b
}

// WithManifestFS sets the embedded filesystem containing manifests.
func (b *Builder) WithManifestFS(fs embed.FS) *Builder {
	b.config.ManifestFS = fs
	return b
}

// WithManifestRoot sets the root path in the embedded filesystem.
func (b *Builder) WithManifestRoot(root string) *Builder {
	b.config.ManifestRoot = root
	return b
}

// WithCustomTemplateFS sets custom HTML templates.
func (b *Builder) WithCustomTemplateFS(fs *embed.FS) *Builder {
	b.config.CustomTemplateFS = fs
	return b
}

// WithDataPath sets the data storage path.
func (b *Builder) WithDataPath(path string) *Builder {
	b.config.DataPath = path
	return b
}

// WithPort sets the HTTP server port.
func (b *Builder) WithPort(port string) *Builder {
	b.config.Port = port
	return b
}

// WithLogRetentionDays sets the log retention period in days.
func (b *Builder) WithLogRetentionDays(days int) *Builder {
	b.config.LogRetentionDays = days
	return b
}

// WithLogCleanupInterval sets the log cleanup interval.
func (b *Builder) WithLogCleanupInterval(interval time.Duration) *Builder {
	b.config.LogCleanupInterval = interval
	return b
}

// WithCRDGroup sets the CRD group name.
func (b *Builder) WithCRDGroup(group string) *Builder {
	b.config.CRDGroup = group
	return b
}

// WithCRDVersion sets the CRD version.
func (b *Builder) WithCRDVersion(version string) *Builder {
	b.config.CRDVersion = version
	return b
}

// WithCRDResource sets the CRD resource name.
func (b *Builder) WithCRDResource(resource string) *Builder {
	b.config.CRDResource = resource
	return b
}

// Build returns the configured Config and validates it.
// Returns an error if validation fails.
func (b *Builder) Build() (framework.Config, error) {
	if err := b.config.Validate(); err != nil {
		return framework.Config{}, err
	}
	return b.config, nil
}

// MustBuild returns the configured Config and panics if validation fails.
func (b *Builder) MustBuild() framework.Config {
	cfg, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("invalid configuration: %v", err))
	}
	return cfg
}

