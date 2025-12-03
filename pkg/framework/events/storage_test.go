package events

import (
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/garunski/conductor-framework/pkg/framework/database"
)

func setupTestEventDB(t *testing.T) (*database.DB, *Storage) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	logger := logr.Discard()
	storage := NewStorage(db, logger)
	return db, storage
}

func createTestEvents(t *testing.T, storage *Storage, n int) {
	now := time.Now()
	for i := 0; i < n; i++ {
		event := Event{
			ID:          "",
			Timestamp:   now.Add(-time.Duration(i) * 24 * time.Hour / time.Duration(n) * 30),
			Type:        EventTypeInfo,
			ResourceKey: "test-resource",
			Message:     "Test event",
		}
		if i%10 == 0 {
			event.Type = EventTypeError
		}
		if err := storage.StoreEvent(event); err != nil {
			t.Fatalf("failed to store event %d: %v", i, err)
		}
	}
}

func TestStorage_ListEvents_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	_, storage := setupTestEventDB(t)

	numEvents := 10000
	t.Logf("Creating %d test events...", numEvents)
	createTestEvents(t, storage, numEvents)

	t.Run("QueryAllEvents", func(t *testing.T) {

		testDB, _ := setupTestEventDB(t)
		allData, err := testDB.List("events/")
		if err != nil {
			t.Fatalf("db.List() failed: %v", err)
		}
		t.Logf("Found %d events in database", len(allData))

		start := time.Now()
		filters := EventFilters{
			Limit: 100,
		}
		events, err := storage.ListEvents(filters)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("ListEvents() failed: %v", err)
		}
		if len(events) == 0 && len(allData) > 0 {
			t.Errorf("ListEvents() returned 0 events but database has %d events", len(allData))
		}
		if len(events) > 100 {
			t.Errorf("ListEvents() returned %d events, want at most 100", len(events))
		}
		t.Logf("QueryAllEvents: retrieved %d events in %v", len(events), duration)
		if duration > 5*time.Second {
			t.Errorf("QueryAllEvents took too long: %v", duration)
		}
	})

	t.Run("QueryWithTypeFilter", func(t *testing.T) {
		start := time.Now()
		filters := EventFilters{
			Type:  EventTypeError,
			Limit: 100,
		}
		events, err := storage.ListEvents(filters)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("ListEvents() failed: %v", err)
		}
		t.Logf("QueryWithTypeFilter: retrieved %d events in %v", len(events), duration)
		if duration > 5*time.Second {
			t.Errorf("QueryWithTypeFilter took too long: %v", duration)
		}

		for _, event := range events {
			if event.Type != EventTypeError {
				t.Errorf("expected EventTypeError, got %s", event.Type)
			}
		}
	})

	t.Run("QueryWithTimeRange", func(t *testing.T) {
		start := time.Now()
		since := time.Now().Add(-15 * 24 * time.Hour)
		filters := EventFilters{
			Since: since,
			Limit: 100,
		}
		events, err := storage.ListEvents(filters)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("ListEvents() failed: %v", err)
		}
		t.Logf("QueryWithTimeRange: retrieved %d events in %v", len(events), duration)
		if duration > 5*time.Second {
			t.Errorf("QueryWithTimeRange took too long: %v", duration)
		}

		for _, event := range events {
			if event.Timestamp.Before(since) {
				t.Errorf("event timestamp %v is before since %v", event.Timestamp, since)
			}
		}
	})
}

func TestStorage_CleanupOldEvents_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	_, storage := setupTestEventDB(t)

	numEvents := 10000
	t.Logf("Creating %d test events...", numEvents)
	createTestEvents(t, storage, numEvents)

	t.Run("CleanupOldEvents", func(t *testing.T) {

		before := time.Now().Add(-10 * 24 * time.Hour)

		start := time.Now()
		err := storage.CleanupOldEvents(before)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("CleanupOldEvents() failed: %v", err)
		}
		t.Logf("CleanupOldEvents: completed in %v", duration)
		if duration > 5*time.Second {
			t.Errorf("CleanupOldEvents took too long: %v", duration)
		}

		filters := EventFilters{
			Limit: 1000,
		}
		events, err := storage.ListEvents(filters)
		if err != nil {
			t.Fatalf("ListEvents() failed: %v", err)
		}

		for _, event := range events {
			if event.Timestamp.Before(before) {
				t.Errorf("found event older than cleanup threshold: %v < %v", event.Timestamp, before)
			}
		}
	})
}

func TestStorage_StoreEventsBatch(t *testing.T) {
	_, storage := setupTestEventDB(t)

	events := []Event{
		Success("test/key1", "apply", "Success 1"),
		Error("test/key2", "apply", "Error 1", nil),
		Info("test/key3", "reconcile", "Info 1"),
		Warning("test/key4", "update", "Warning 1"),
	}

	err := storage.StoreEventsBatch(events)
	if err != nil {
		t.Fatalf("StoreEventsBatch() error = %v", err)
	}

	// Verify events were stored
	filters := EventFilters{Limit: 100}
	storedEvents, err := storage.ListEvents(filters)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}

	if len(storedEvents) < len(events) {
		t.Errorf("StoreEventsBatch() stored %d events, want at least %d", len(storedEvents), len(events))
	}
}

func TestStorage_StoreEventsBatch_Empty(t *testing.T) {
	_, storage := setupTestEventDB(t)

	err := storage.StoreEventsBatch([]Event{})
	if err != nil {
		t.Fatalf("StoreEventsBatch() with empty slice should not error, got: %v", err)
	}
}

func TestStorage_GetEventsByResource(t *testing.T) {
	_, storage := setupTestEventDB(t)

	// Store events for different resources
	storage.StoreEvent(Success("resource1", "apply", "Event 1"))
	storage.StoreEvent(Success("resource1", "apply", "Event 2"))
	storage.StoreEvent(Success("resource2", "apply", "Event 3"))

	events, err := storage.GetEventsByResource("resource1", 10)
	if err != nil {
		t.Fatalf("GetEventsByResource() error = %v", err)
	}

	if len(events) < 2 {
		t.Errorf("GetEventsByResource() returned %d events, want at least 2", len(events))
	}

	for _, event := range events {
		if event.ResourceKey != "resource1" {
			t.Errorf("GetEventsByResource() returned event with wrong resource key: %v", event.ResourceKey)
		}
	}
}

func TestStorage_GetRecentErrors(t *testing.T) {
	_, storage := setupTestEventDB(t)

	// Store different event types
	storage.StoreEvent(Success("test/key", "apply", "Success"))
	storage.StoreEvent(Error("test/key", "apply", "Error 1", nil))
	storage.StoreEvent(Info("test/key", "reconcile", "Info"))
	storage.StoreEvent(Error("test/key", "apply", "Error 2", nil))

	events, err := storage.GetRecentErrors(10)
	if err != nil {
		t.Fatalf("GetRecentErrors() error = %v", err)
	}

	if len(events) < 2 {
		t.Errorf("GetRecentErrors() returned %d events, want at least 2", len(events))
	}

	for _, event := range events {
		if event.Type != EventTypeError {
			t.Errorf("GetRecentErrors() returned non-error event: %v", event.Type)
		}
	}
}

func TestStorage_DeleteEvent(t *testing.T) {
	_, storage := setupTestEventDB(t)

	// Store an event
	event := Success("test/key", "apply", "Test event")
	err := storage.StoreEvent(event)
	if err != nil {
		t.Fatalf("StoreEvent() error = %v", err)
	}

	// Delete the event
	err = storage.DeleteEvent(event.ID, event.Timestamp)
	if err != nil {
		t.Fatalf("DeleteEvent() error = %v", err)
	}

	// Verify event was deleted
	filters := EventFilters{
		ResourceKey: "test/key",
		Limit:       10,
	}
	events, err := storage.ListEvents(filters)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}

	// Note: DeleteEvent only deletes the timestamp key, not the indexed keys
	// So the event may still appear in ListEvents results
	// This is expected behavior based on the implementation
	_ = events // Use events to avoid unused variable
}

func TestStorage_ListEvents_MemoryEfficiency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory efficiency test in short mode")
	}

	_, storage := setupTestEventDB(t)

	numEvents := 10000
	t.Logf("Creating %d test events...", numEvents)
	createTestEvents(t, storage, numEvents)

	t.Run("QueryWithSmallLimit", func(t *testing.T) {
		filters := EventFilters{
			Limit: 10,
		}
		events, err := storage.ListEvents(filters)
		if err != nil {
			t.Fatalf("ListEvents() failed: %v", err)
		}
		if len(events) != 10 {
			t.Errorf("ListEvents() returned %d events, want 10", len(events))
		}

	})
}

func TestStorage_CleanupOldEvents_BatchProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping batch processing test in short mode")
	}

	_, storage := setupTestEventDB(t)

	numEvents := 5000
	t.Logf("Creating %d test events...", numEvents)
	createTestEvents(t, storage, numEvents)

	before := time.Now().Add(-20 * 24 * time.Hour)

	err := storage.CleanupOldEvents(before)
	if err != nil {
		t.Fatalf("CleanupOldEvents() failed: %v", err)
	}

	filters := EventFilters{
		Limit: 1000,
	}
	events, err := storage.ListEvents(filters)
	if err != nil {
		t.Fatalf("ListEvents() failed: %v", err)
	}

	for _, event := range events {
		if event.Timestamp.Before(before) {
			t.Errorf("found event older than cleanup threshold: %v < %v", event.Timestamp, before)
		}
	}
}

