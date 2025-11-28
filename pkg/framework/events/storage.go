package events

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/garunski/conductor-framework/pkg/framework/database"
)

type Storage struct {
	db     *database.DB
	logger logr.Logger
}

func NewStorage(db *database.DB, logger logr.Logger) *Storage {
	return &Storage{
		db:     db,
		logger: logger,
	}
}

func (s *Storage) StoreEvent(event Event) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	timestampKey := fmt.Sprintf("events/%020d/%s", event.Timestamp.UnixNano(), event.ID)
	if err := s.db.Set(timestampKey, data); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	if event.ResourceKey != "" {
		resourceKey := fmt.Sprintf("events/by-resource/%s/%020d/%s", event.ResourceKey, event.Timestamp.UnixNano(), event.ID)
		if err := s.db.Set(resourceKey, data); err != nil {
			s.logger.Error(err, "failed to store resource index", "key", resourceKey)

		}
	}

	typeKey := fmt.Sprintf("events/by-type/%s/%020d/%s", event.Type, event.Timestamp.UnixNano(), event.ID)
	if err := s.db.Set(typeKey, data); err != nil {
		s.logger.Error(err, "failed to store type index", "key", typeKey)

	}

	return nil
}

func (s *Storage) StoreEventsBatch(events []Event) error {
	if len(events) == 0 {
		return nil
	}

	batchItems := make(map[string][]byte)

	for _, event := range events {

		if event.ID == "" {
			event.ID = uuid.New().String()
		}
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now()
		}

		data, err := json.Marshal(event)
		if err != nil {
			s.logger.Error(err, "failed to marshal event in batch", "eventID", event.ID)
			continue
		}

		timestampKey := fmt.Sprintf("events/%020d/%s", event.Timestamp.UnixNano(), event.ID)
		batchItems[timestampKey] = data

		if event.ResourceKey != "" {
			resourceKey := fmt.Sprintf("events/by-resource/%s/%020d/%s", event.ResourceKey, event.Timestamp.UnixNano(), event.ID)
			batchItems[resourceKey] = data
		}

		typeKey := fmt.Sprintf("events/by-type/%s/%020d/%s", event.Type, event.Timestamp.UnixNano(), event.ID)
		batchItems[typeKey] = data
	}

	if err := s.db.BatchSet(batchItems); err != nil {
		return fmt.Errorf("failed to store events batch: %w", err)
	}

	return nil
}

func (s *Storage) ListEvents(filters EventFilters) ([]Event, error) {
	var prefix string

	if filters.ResourceKey != "" {
		prefix = fmt.Sprintf("events/by-resource/%s/", filters.ResourceKey)
	} else if filters.Type != "" {
		prefix = fmt.Sprintf("events/by-type/%s/", filters.Type)
	} else {
		prefix = "events/"
	}

	allItems, err := s.db.List(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var events []Event
	for _, data := range allItems {
		var event Event
		if err := json.Unmarshal(data, &event); err != nil {
			s.logger.Error(err, "failed to unmarshal event", "key", prefix)
			continue
		}

		if filters.ResourceKey != "" && event.ResourceKey != filters.ResourceKey {
			continue
		}
		if filters.Type != "" && event.Type != filters.Type {
			continue
		}
		if !filters.Since.IsZero() && event.Timestamp.Before(filters.Since) {
			continue
		}
		if !filters.Until.IsZero() && event.Timestamp.After(filters.Until) {
			continue
		}

		events = append(events, event)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(events) {
		return []Event{}, nil
	}
	if offset > 0 {
		events = events[offset:]
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 100
	}
	if len(events) > limit {
		events = events[:limit]
	}

	return events, nil
}

func (s *Storage) GetEventsByResource(key string, limit int) ([]Event, error) {
	filters := EventFilters{
		ResourceKey: key,
		Limit:       limit,
	}
	return s.ListEvents(filters)
}

func (s *Storage) GetRecentErrors(limit int) ([]Event, error) {
	filters := EventFilters{
		Type:  EventTypeError,
		Limit: limit,
	}
	return s.ListEvents(filters)
}

func (s *Storage) CleanupOldEvents(before time.Time) error {
	const batchSize = 1000
	deletedCount := 0
	beforeTimestamp := before.UnixNano()

	allItems, err := s.db.List("events/")
	if err != nil {
		return fmt.Errorf("failed to list events for cleanup: %w", err)
	}

	type eventKey struct {
		key   string
		event Event
	}
	var oldEvents []eventKey

	for key, data := range allItems {

		if strings.Contains(key, "/by-resource/") || strings.Contains(key, "/by-type/") {
			continue
		}

		var event Event
		if err := json.Unmarshal(data, &event); err != nil {

			oldEvents = append(oldEvents, eventKey{key: key})
			continue
		}

		eventTimestamp := event.Timestamp.UnixNano()
		if eventTimestamp < beforeTimestamp {
			oldEvents = append(oldEvents, eventKey{key: key, event: event})
		}
	}

	totalProcessed := len(oldEvents)
	for i := 0; i < len(oldEvents); i += batchSize {
		end := i + batchSize
		if end > len(oldEvents) {
			end = len(oldEvents)
		}

		batch := oldEvents[i:end]
		keysToDelete := make([]string, 0, len(batch)*3)

		for _, ek := range batch {
			if ek.event.ID == "" {

				keysToDelete = append(keysToDelete, ek.key)
				continue
			}

			eventTimestamp := ek.event.Timestamp.UnixNano()
			timestampKey := fmt.Sprintf("events/%020d/%s", eventTimestamp, ek.event.ID)
			keysToDelete = append(keysToDelete, timestampKey)

			if ek.event.ResourceKey != "" {
				resourceKey := fmt.Sprintf("events/by-resource/%s/%020d/%s", ek.event.ResourceKey, eventTimestamp, ek.event.ID)
				keysToDelete = append(keysToDelete, resourceKey)
			}

			typeKey := fmt.Sprintf("events/by-type/%s/%020d/%s", ek.event.Type, eventTimestamp, ek.event.ID)
			keysToDelete = append(keysToDelete, typeKey)
		}

		if len(keysToDelete) > 0 {
			if err := s.db.BatchDelete(keysToDelete); err != nil {
				s.logger.Error(err, "failed to batch delete events", "count", len(keysToDelete))

				for _, key := range keysToDelete {
					if err := s.db.Delete(key); err != nil {

						if strings.Contains(key, "/by-resource/") || strings.Contains(key, "/by-type/") {
							s.logger.V(1).Info("failed to delete event index entry (non-critical)", "key", key, "error", err)
						} else {
							s.logger.Error(err, "failed to delete event", "key", key)
						}
					} else {
						deletedCount++
					}
				}
			} else {
				deletedCount += len(keysToDelete)
			}
		}

		if (i+batchSize)%10000 == 0 || end == len(oldEvents) {
			s.logger.Info("Cleanup in progress", "processed", end, "deleted", deletedCount)
		}
	}

	s.logger.Info("Cleaned up old events", "deleted", deletedCount, "processed", totalProcessed, "before", before)
	return nil
}

func (s *Storage) DeleteEvent(id string, timestamp time.Time) error {

	timestampKey := fmt.Sprintf("events/%020d/%s", timestamp.UnixNano(), id)
	return s.db.Delete(timestampKey)
}

