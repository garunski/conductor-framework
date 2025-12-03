package testing

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"github.com/garunski/conductor-framework/pkg/framework/database"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"github.com/garunski/conductor-framework/pkg/framework/index"
	"github.com/garunski/conductor-framework/pkg/framework/store"
)

// NewTestLogger creates a test logger
func NewTestLogger() logr.Logger {
	zapLog, _ := zap.NewDevelopment()
	return zapr.NewLogger(zapLog)
}

// NewTestDB creates a test database
func NewTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	return db
}

// NewTestIndex creates a test index
func NewTestIndex() *index.ManifestIndex {
	return index.NewIndex()
}

// NewTestEventStore creates a test event store
func NewTestEventStore(t *testing.T) events.EventStorage {
	db := NewTestDB(t)
	return events.NewStorage(db, NewTestLogger())
}

// NewTestManifestStore creates a test manifest store
func NewTestManifestStore(t *testing.T) store.ManifestStore {
	db := NewTestDB(t)
	idx := NewTestIndex()
	return store.NewManifestStore(db, idx, NewTestLogger())
}

