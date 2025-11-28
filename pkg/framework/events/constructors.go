package events

import "time"

func Success(resourceKey, operation, message string) Event {
	return Event{
		Type:        EventTypeSuccess,
		ResourceKey: resourceKey,
		Message:     message,
		Timestamp:   time.Now(),
		Details: map[string]interface{}{
			"operation": operation,
		},
	}
}

func Error(resourceKey, operation, message string, err error) Event {
	event := Event{
		Type:        EventTypeError,
		ResourceKey: resourceKey,
		Message:     message,
		Timestamp:   time.Now(),
		Details: map[string]interface{}{
			"operation": operation,
		},
	}
	if err != nil {
		event.Error = err.Error()
	}
	return event
}

func Info(resourceKey, operation, message string) Event {
	return Event{
		Type:        EventTypeInfo,
		ResourceKey: resourceKey,
		Message:     message,
		Timestamp:   time.Now(),
		Details: map[string]interface{}{
			"operation": operation,
		},
	}
}

func Warning(resourceKey, operation, message string) Event {
	return Event{
		Type:        EventTypeWarning,
		ResourceKey: resourceKey,
		Message:     message,
		Timestamp:   time.Now(),
		Details: map[string]interface{}{
			"operation": operation,
		},
	}
}

