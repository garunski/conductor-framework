package events

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/garunski/conductor-framework/pkg/framework/database"
)

func TestStoreEventSafe(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	logger := logr.Discard()
	storage := NewStorage(db, logger)

	event := Success("test/key", "apply", "Test message")
	StoreEventSafe(storage, logger, event)

	// Verify event was stored
	filters := EventFilters{
		ResourceKey: "test/key",
		Limit:       10,
	}
	events, err := storage.ListEvents(filters)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}

	if len(events) == 0 {
		t.Error("StoreEventSafe() did not store event")
	}

	found := false
	for _, e := range events {
		if e.ResourceKey == "test/key" && e.Message == "Test message" {
			found = true
			break
		}
	}

	if !found {
		t.Error("StoreEventSafe() stored event not found in list")
	}
}

func TestStoreEventSafe_NilStorage(t *testing.T) {
	logger := logr.Discard()
	event := Success("test/key", "apply", "Test message")

	// Should not panic with nil storage
	StoreEventSafe(nil, logger, event)
}

func TestStoreEventSafe_StorageError(t *testing.T) {
	// Create a storage that will fail (closed DB)
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	logger := logr.Discard()
	storage := NewStorage(db, logger)
	
	// Close the DB to cause storage errors
	db.Close()

	event := Success("test/key", "apply", "Test message")
	// Should not panic, just log the error
	StoreEventSafe(storage, logger, event)
}

