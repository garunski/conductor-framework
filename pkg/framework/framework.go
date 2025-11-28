package framework

import (
	"context"
	"embed"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
	"github.com/garunski/conductor-framework/pkg/framework/manifest"
	"github.com/garunski/conductor-framework/pkg/framework/reconciler"
	"github.com/garunski/conductor-framework/pkg/framework/server"
	"k8s.io/client-go/dynamic"
)

// Config holds all framework configuration
type Config struct {
	// Application metadata
	AppName    string
	AppVersion string

	// Manifest configuration
	ManifestFS      embed.FS
	ManifestRoot    string
	CustomTemplateFS *embed.FS // Optional custom templates
	TemplateFuncs   template.FuncMap // Optional custom template functions

	// Storage configuration
	DataPath string

	// Server configuration
	Port string

	// Logging configuration
	LogRetentionDays  int
	LogCleanupInterval time.Duration

	// CRD configuration
	CRDGroup         string
	CRDVersion       string
	CRDResource      string
}

// DefaultConfig returns a Config with default values
func DefaultConfig() Config {
	return Config{
		AppName:            "conductor",
		AppVersion:         getEnvOrDefault("VERSION", "dev"),
		ManifestRoot:       "manifests",
		DataPath:           getEnvOrDefault("BADGER_DATA_PATH", "/data/badger"),
		Port:               getEnvOrDefault("PORT", "8081"),
		LogRetentionDays:   parseIntOrDefault("LOG_RETENTION_DAYS", 7),
		LogCleanupInterval: parseDurationOrDefault("LOG_CLEANUP_INTERVAL", 1*time.Hour),
		CRDGroup:           crd.DefaultCRDGroup,
		CRDVersion:         crd.DefaultCRDVersion,
		CRDResource:        crd.DefaultCRDResource,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.AppName == "" {
		return fmt.Errorf("AppName cannot be empty")
	}
	if c.DataPath == "" {
		return fmt.Errorf("DataPath cannot be empty")
	}
	if c.Port == "" {
		return fmt.Errorf("Port cannot be empty")
	}
	if c.LogRetentionDays < 0 {
		return fmt.Errorf("LogRetentionDays cannot be negative")
	}
	if c.LogCleanupInterval <= 0 {
		return fmt.Errorf("LogCleanupInterval must be positive")
	}
	return nil
}

// Run starts the framework with the given configuration
// It handles the complete lifecycle: initialization, startup, and shutdown
func Run(ctx context.Context, cfg Config) error {
	// Initialize logger
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	logger := zapr.NewLogger(zapLog)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	logger.Info("Starting framework", "appName", cfg.AppName, "version", cfg.AppVersion)

	// Load manifests with optional parameter templating
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var manifests map[string][]byte
	var parameterGetter manifest.ParameterGetter

	// Always attempt to create parameter client for manifest loading
	// If Kubernetes is unavailable, fall back to default parameters
	logger.Info("Setting up Kubernetes client for manifest loading")
	kubeConfig, err := reconciler.GetKubernetesConfig()
	if err != nil {
		logger.Info("Kubernetes config not available, using default parameters for template rendering", "error", err)
		// Use nil parameterGetter which will use defaults in manifest loader
		parameterGetter = nil
	} else {
		dynamicClient, err := dynamic.NewForConfig(kubeConfig)
		if err != nil {
			logger.Info("Failed to create dynamic client, using default parameters for template rendering", "error", err)
			// Use nil parameterGetter which will use defaults in manifest loader
			parameterGetter = nil
		} else {
			// Create temporary CRD client for manifest loading
			parameterClient := crd.NewClient(dynamicClient, logger, cfg.CRDGroup, cfg.CRDVersion, cfg.CRDResource)
			
			// Create parameter getter function
			defaultNamespace := "default"
			parameterGetter = func(ctx context.Context, serviceName string) (*crd.ParameterSet, error) {
				return parameterClient.GetMergedParameters(ctx, serviceName, defaultNamespace)
			}
		}
	}

	logger.Info("Loading embedded manifests with template rendering")
	manifests, err = manifest.LoadEmbeddedManifests(cfg.ManifestFS, cfg.ManifestRoot, ctx, parameterGetter, cfg.TemplateFuncs)
	if err != nil {
		return fmt.Errorf("failed to load embedded manifests: %w", err)
	}
	logger.Info("Loaded manifests", "count", len(manifests))

	// Convert Config to server.Config
	serverCfg := &server.Config{
		AppName:            cfg.AppName,
		AppVersion:         cfg.AppVersion,
		DataPath:           cfg.DataPath,
		Port:               cfg.Port,
		LogRetentionDays:   cfg.LogRetentionDays,
		LogCleanupInterval: cfg.LogCleanupInterval,
		CRDGroup:           cfg.CRDGroup,
		CRDVersion:         cfg.CRDVersion,
		CRDResource:        cfg.CRDResource,
		CustomTemplateFS:   cfg.CustomTemplateFS,
	}

	// Create server with pre-loaded manifests
	srv, err := server.NewServer(serverCfg, logger, manifests)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	defer func() {
		if err := srv.Close(); err != nil {
			logger.Error(err, "failed to close server")
		}
	}()

	// Start server
	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Wait for shutdown
	if err := srv.WaitForShutdown(); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	return nil
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func parseIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		// Try to parse as duration first (e.g., "7d")
		if d, err := time.ParseDuration(value + "d"); err == nil {
			return int(d.Hours() / 24)
		}
		// Try to parse as int
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}

