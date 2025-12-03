package events

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/garunski/conductor-framework/pkg/framework/database"
	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
)

type Storage struct {
	db     *database.DB
	logger logr.Logger
}

func NewStorage(db *database.DB, logger logr.Logger) EventStorage {
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
		return apperrors.WrapStorage(err, "failed to marshal event")
	}

	timestampKey := fmt.Sprintf("events/%020d/%s", event.Timestamp.UnixNano(), event.ID)
	if err := s.db.Set(timestampKey, data); err != nil {
		return apperrors.WrapStorage(err, "failed to store event")
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
		return apperrors.WrapStorage(err, "failed to store events batch")
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
		return nil, apperrors.WrapStorage(err, "failed to list events")
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

