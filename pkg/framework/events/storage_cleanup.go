package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
)

func (s *Storage) CleanupOldEvents(before time.Time) error {
	deletedCount := 0
	beforeTimestamp := before.UnixNano()

	allItems, err := s.db.List("events/")
	if err != nil {
		return apperrors.WrapStorage(err, "failed to list events for cleanup")
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
	for i := 0; i < len(oldEvents); i += DefaultBatchSize {
		end := i + DefaultBatchSize
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

		if (i+DefaultBatchSize)%10000 == 0 || end == len(oldEvents) {
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

