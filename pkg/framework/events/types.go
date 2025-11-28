package events

import "time"

type EventType string

const (
	EventTypeError   EventType = "error"
	EventTypeSuccess EventType = "success"
	EventTypeInfo    EventType = "info"
	EventTypeWarning EventType = "warning"
)

type Event struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        EventType              `json:"type"`
	ResourceKey string                 `json:"resourceKey,omitempty"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type EventFilters struct {
	ResourceKey string
	Type        EventType
	Since       time.Time
	Until       time.Time
	Limit       int
	Offset      int
}

