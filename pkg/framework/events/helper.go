package events

import "github.com/go-logr/logr"

func StoreEventSafe(storage EventStorage, logger logr.Logger, event Event) {
	if storage == nil {
		return
	}
	if err := storage.StoreEvent(event); err != nil {
		logger.V(1).Info("failed to store event",
			"error", err,
			"type", event.Type,
			"resourceKey", event.ResourceKey,
			"message", event.Message)
	}
}

