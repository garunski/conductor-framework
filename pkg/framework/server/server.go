package server

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/garunski/conductor-framework/pkg/framework/api"
	"github.com/garunski/conductor-framework/pkg/framework/crd"
	"github.com/garunski/conductor-framework/pkg/framework/database"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"github.com/garunski/conductor-framework/pkg/framework/index"
	"github.com/garunski/conductor-framework/pkg/framework/reconciler"
	"github.com/garunski/conductor-framework/pkg/framework/store"
)

// Config holds server configuration
type Config struct {
	AppName            string
	AppVersion         string
	DataPath           string
	Port               string
	ReconcileInterval  time.Duration
	AutoDeploy         bool
	LogRetentionDays   int
	LogCleanupInterval time.Duration
	CRDGroup           string
	CRDVersion         string
	CRDResource        string
	EnableParameters   bool
	CustomTemplateFS   *embed.FS // Optional custom templates
}

type Server struct {
	config          *Config
	logger          logr.Logger
	db              *database.DB
	index           *index.ManifestIndex
	eventStore      *events.Storage
	reconciler      *reconciler.Reconciler
	handler         *api.Handler
	httpServer      *http.Server
	reconcileCh     chan string
	parameterClient *crd.Client
}

// NewServer creates a new server instance
// manifests should be pre-loaded before calling this function
func NewServer(cfg *Config, logger logr.Logger, manifests map[string][]byte) (*Server, error) {
	ctx := context.Background()

	logger.Info("Setting up Kubernetes client")
	kubeConfig, err := reconciler.GetKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Initialize CRD parameter client if enabled
	var parameterClient *crd.Client
	if cfg.EnableParameters {
		parameterClient = crd.NewClient(dynamicClient, logger, cfg.CRDGroup, cfg.CRDVersion, cfg.CRDResource)
		logger.Info("CRD parameter client initialized", "group", cfg.CRDGroup, "version", cfg.CRDVersion, "resource", cfg.CRDResource)

		// Get or create default DeploymentParameters instance
		defaultNamespace := "default"
		defaultParams, err := parameterClient.Get(ctx, crd.DefaultName, defaultNamespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get default DeploymentParameters: %w", err)
		}

		if defaultParams == nil {
			logger.Info("Creating default DeploymentParameters instance")
			defaultParams = &crd.DeploymentParameters{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crd.DefaultName,
					Namespace: defaultNamespace,
				},
				Spec: crd.DeploymentParametersSpec{
					Global: &crd.ParameterSet{
						Namespace:  "default",
						NamePrefix: "",
						Replicas:   int32Ptr(1),
					},
				},
			}
			if err := parameterClient.Create(ctx, defaultParams); err != nil {
				logger.Error(err, "failed to create default DeploymentParameters, continuing without it")
			} else {
				logger.Info("Created default DeploymentParameters instance")
			}
		}
	} else {
		logger.Info("CRD parameters disabled, skipping parameter client initialization")
	}

	logger.Info("Opening BadgerDB", "path", cfg.DataPath)
	db, err := database.NewDB(cfg.DataPath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}

	logger.Info("Loading DB overrides")
	dbOverrides, err := db.List("")
	if err != nil {
		return nil, fmt.Errorf("failed to load DB overrides: %w", err)
	}
	logger.Info("Loaded DB overrides", "count", len(dbOverrides))

	idx := index.NewIndex()
	idx.Merge(manifests, dbOverrides)

	eventStore := events.NewStorage(db, logger)
	logger.Info("Event storage initialized")

	manifestStore := store.NewManifestStore(db, idx, logger)

	// Use Config.AppName for reconciler field manager
	appName := cfg.AppName
	if appName == "" {
		appName = "conductor"
	}

	rec, err := reconciler.NewReconciler(
		clientset,
		dynamicClient,
		manifestStore,
		logger,
		eventStore,
		appName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reconciler: %w", err)
	}

	reconcileCh := make(chan string, 100)

	handler, err := api.NewHandler(
		manifestStore,
		eventStore,
		logger,
		reconcileCh,
		rec,
		cfg.AppName,
		cfg.AppVersion,
		parameterClient,
		cfg.CustomTemplateFS,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	router := handler.SetupRoutes()
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	return &Server{
		config:          cfg,
		logger:          logger,
		db:              db,
		index:           idx,
		eventStore:      eventStore,
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

