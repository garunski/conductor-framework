package server

import (
	"fmt"

	"github.com/go-logr/logr"

	"github.com/garunski/conductor-framework/pkg/framework/database"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"github.com/garunski/conductor-framework/pkg/framework/index"
	"github.com/garunski/conductor-framework/pkg/framework/store"
)

// StorageComponents holds all storage-related components
type StorageComponents struct {
	DB          *database.DB
	Index       *index.ManifestIndex
	EventStore  events.EventStorage
	ManifestStore store.ManifestStore
}

// NewStorageComponents creates and initializes all storage components.
// It opens the database, loads overrides, creates the index, event store, and manifest store.
func NewStorageComponents(cfg *Config, logger logr.Logger, manifests map[string][]byte) (*StorageComponents, error) {
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

	return &StorageComponents{
		DB:           db,
		Index:        idx,
		EventStore:   eventStore,
		ManifestStore: manifestStore,
	}, nil
}

