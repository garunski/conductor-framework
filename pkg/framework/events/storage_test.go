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

