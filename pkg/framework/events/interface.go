package events

import "time"

// EventStorage defines the interface for event storage operations.
// This interface allows for better testability and reduced coupling.
type EventStorage interface {
	// StoreEvent stores a single event
	StoreEvent(event Event) error

	// StoreEventsBatch stores multiple events in a batch operation
	StoreEventsBatch(events []Event) error

	// ListEvents lists events matching the provided filters
	ListEvents(filters EventFilters) ([]Event, error)

	// GetEventsByResource retrieves events for a specific resource key
	GetEventsByResource(key string, limit int) ([]Event, error)

	// GetRecentErrors retrieves recent error events
	GetRecentErrors(limit int) ([]Event, error)

	// CleanupOldEvents removes events older than the specified time
	CleanupOldEvents(before time.Time) error

	// DeleteEvent deletes a specific event by ID and timestamp
	DeleteEvent(id string, timestamp time.Time) error
}

// Ensure *Storage implements EventStorage interface
var _ EventStorage = (*Storage)(nil)

