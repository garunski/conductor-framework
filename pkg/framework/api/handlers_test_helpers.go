package api

import (
	"embed"
	"testing"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
	"github.com/garunski/conductor-framework/pkg/framework/database"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"github.com/garunski/conductor-framework/pkg/framework/index"
	"github.com/garunski/conductor-framework/pkg/framework/reconciler"
	"github.com/garunski/conductor-framework/pkg/framework/store"
)

// Core test handler setup functions are defined here.
// Domain-specific helpers are in:
// - handlers_test_helpers_services.go (services/reconciler helpers)
// - handlers_test_helpers_parameters.go (parameters helpers)
// - handlers_test_helpers_web.go (web helpers)

func newTestHandler(t *testing.T, opts ...testHandlerOption) (*Handler, error) {
	t.Helper()
	logger := logr.Discard()
	
	// Default options
	cfg := testHandlerConfig{
		appName:    "test-app",
		version:    "test-version",
		logger:     logger,
		reconcileCh: make(chan string, 100),
	}
	
	// Apply options
	for _, opt := range opts {
		opt(&cfg)
	}
	
	// Create database if not provided
	if cfg.db == nil {
		db, err := database.NewTestDB(t)
		if err != nil {
			return nil, err
		}
		cfg.db = db
	}
	
	// Create index and store if not provided
	if cfg.store == nil {
		idx := index.NewIndex()
		cfg.store = store.NewManifestStore(cfg.db, idx, logger)
	}
	
	// Create event store if not provided and not explicitly set to nil
	if cfg.eventStore == nil && !cfg.eventStoreSet {
		cfg.eventStore = events.NewStorage(cfg.db, logger)
	}
	
	// Create parameter client if not provided
	if cfg.parameterClient == nil {
		scheme := runtime.NewScheme()
		corev1.AddToScheme(scheme)
		appsv1.AddToScheme(scheme)
		dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
		cfg.parameterClient = crd.NewClient(dynamicClient, logger, "conductor.io", "v1alpha1", "deploymentparameters")
	}
	
	var emptyFS embed.FS
	return NewHandler(cfg.store, cfg.eventStore, cfg.logger, cfg.reconcileCh, cfg.reconciler, cfg.appName, cfg.version, cfg.parameterClient, nil, emptyFS, "")
}

type testHandlerConfig struct {
	appName         string
	version         string
	logger          logr.Logger
	reconcileCh     chan string
	reconciler      reconciler.Reconciler
	db              *database.DB
	store           store.ManifestStore
	eventStore      events.EventStorage
	eventStoreSet   bool // Track if eventStore was explicitly set (even if nil)
	parameterClient *crd.Client
}

type testHandlerOption func(*testHandlerConfig)

func WithTestReconciler(rec reconciler.Reconciler) testHandlerOption {
	return func(cfg *testHandlerConfig) {
		cfg.reconciler = rec
	}
}

func WithTestEventStore(eventStore events.EventStorage) testHandlerOption {
	return func(cfg *testHandlerConfig) {
		cfg.eventStore = eventStore
		cfg.eventStoreSet = true
	}
}

func WithNilReconciler() testHandlerOption {
	return func(cfg *testHandlerConfig) {
		cfg.reconciler = nil
	}
}

func WithNilEventStore() testHandlerOption {
	return func(cfg *testHandlerConfig) {
		cfg.eventStore = nil
		cfg.eventStoreSet = true
	}
}

func setupTestHandler(t *testing.T) (*Handler, *database.DB) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("NewTestDB() error = %v", err)
	}
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}
	return handler, db
}


func setupTestHandlerWithEventStore(t *testing.T) (*Handler, *database.DB, events.EventStorage) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("NewTestDB() error = %v", err)
	}
	eventStore := events.NewStorage(db, logr.Discard())
	handler, err := newTestHandler(t, WithTestEventStore(eventStore))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}
	return handler, db, eventStore
}


