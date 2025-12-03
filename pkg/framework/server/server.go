package server

import (
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"

	"github.com/garunski/conductor-framework/pkg/framework/api"
	"github.com/garunski/conductor-framework/pkg/framework/crd"
	"github.com/garunski/conductor-framework/pkg/framework/database"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"github.com/garunski/conductor-framework/pkg/framework/index"
	"github.com/garunski/conductor-framework/pkg/framework/reconciler"
)

// Config holds server configuration
type Config struct {
	AppName            string
	AppVersion         string
	DataPath           string
	Port               string
	LogRetentionDays   int
	LogCleanupInterval time.Duration
	CRDGroup           string
	CRDVersion         string
	CRDResource        string
	CustomTemplateFS   *embed.FS // Optional custom templates
	ManifestFS         embed.FS  // Embedded manifest filesystem
	ManifestRoot       string    // Root path for manifests
}

type Server struct {
	config          *Config
	logger          logr.Logger
	db              *database.DB
	index           *index.ManifestIndex
	eventStore      events.EventStorage
	reconciler      reconciler.Reconciler
	handler         *api.Handler
	httpServer      *http.Server
	reconcileCh     chan string
	parameterClient *crd.Client
}

// NewServer creates a new server instance
// manifests should be pre-loaded before calling this function
func NewServer(cfg *Config, logger logr.Logger, manifests map[string][]byte) (*Server, error) {
	// Create Kubernetes clients
	clientset, dynamicClient, err := NewKubernetesClients(cfg, logger)
	if err != nil {
		return nil, err
	}

	// Create CRD client
	parameterClient, err := NewCRDClient(dynamicClient, cfg, logger)
	if err != nil {
		return nil, err
	}

	// Create storage components
	storage, err := NewStorageComponents(cfg, logger, manifests)
	if err != nil {
		return nil, err
	}

	// Use Config.AppName for reconciler field manager
	appName := cfg.AppName
	if appName == "" {
		appName = "conductor"
	}

	// Create reconciler
	rec, err := reconciler.NewReconciler(
		clientset,
		dynamicClient,
		storage.ManifestStore,
		logger,
		storage.EventStore,
		appName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reconciler: %w", err)
	}

	// Create handler
	reconcileCh := make(chan string, 100)
	handler, err := api.NewHandler(
		storage.ManifestStore,
		storage.EventStore,
		logger,
		reconcileCh,
		rec,
		cfg.AppName,
		cfg.AppVersion,
		parameterClient,
		cfg.CustomTemplateFS,
		cfg.ManifestFS,
		cfg.ManifestRoot,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	// Create HTTP server
	router := handler.SetupRoutes()
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	return &Server{
		config:          cfg,
		logger:          logger,
		db:              storage.DB,
		index:           storage.Index,
		eventStore:      storage.EventStore,
		reconciler:      rec,
		handler:         handler,
		httpServer:      httpServer,
		reconcileCh:     reconcileCh,
		parameterClient: parameterClient,
	}, nil
}

func (s *Server) Close() error {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	return nil
}

func int32Ptr(i int32) *int32 {
	return &i
}

