package events

import (
	"errors"
	"testing"
)

func TestSuccess(t *testing.T) {
	event := Success("test/key", "apply", "Success message")

	if event.Type != EventTypeSuccess {
		t.Errorf("Success() Type = %v, want %v", event.Type, EventTypeSuccess)
	}

	if event.ResourceKey != "test/key" {
		t.Errorf("Success() ResourceKey = %v, want test/key", event.ResourceKey)
	}

	if event.Message != "Success message" {
		t.Errorf("Success() Message = %v, want Success message", event.Message)
	}

	if event.Timestamp.IsZero() {
		t.Error("Success() Timestamp should not be zero")
	}

	if event.Details["operation"] != "apply" {
		t.Errorf("Success() Details[operation] = %v, want apply", event.Details["operation"])
	}
}

func TestError(t *testing.T) {
	testErr := errors.New("test error")
	event := Error("test/key", "apply", "Error message", testErr)

	if event.Type != EventTypeError {
		t.Errorf("Error() Type = %v, want %v", event.Type, EventTypeError)
	}

	if event.ResourceKey != "test/key" {
		t.Errorf("Error() ResourceKey = %v, want test/key", event.ResourceKey)
	}

	if event.Message != "Error message" {
		t.Errorf("Error() Message = %v, want Error message", event.Message)
	}

	if event.Error != "test error" {
		t.Errorf("Error() Error = %v, want test error", event.Error)
	}

	if event.Timestamp.IsZero() {
		t.Error("Error() Timestamp should not be zero")
	}

	if event.Details["operation"] != "apply" {
		t.Errorf("Error() Details[operation] = %v, want apply", event.Details["operation"])
	}
}

func TestError_NilError(t *testing.T) {
	event := Error("test/key", "apply", "Error message", nil)

	if event.Type != EventTypeError {
		t.Errorf("Error() Type = %v, want %v", event.Type, EventTypeError)
	}

	if event.Error != "" {
		t.Errorf("Error() Error = %v, want empty string when err is nil", event.Error)
	}
}

func TestInfo(t *testing.T) {
	event := Info("test/key", "reconcile", "Info message")

	if event.Type != EventTypeInfo {
		t.Errorf("Info() Type = %v, want %v", event.Type, EventTypeInfo)
	}

	if event.ResourceKey != "test/key" {
		t.Errorf("Info() ResourceKey = %v, want test/key", event.ResourceKey)
	}

	if event.Message != "Info message" {
		t.Errorf("Info() Message = %v, want Info message", event.Message)
	}

	if event.Timestamp.IsZero() {
		t.Error("Info() Timestamp should not be zero")
	}

	if event.Details["operation"] != "reconcile" {
		t.Errorf("Info() Details[operation] = %v, want reconcile", event.Details["operation"])
	}
}

func TestWarning(t *testing.T) {
	event := Warning("test/key", "update", "Warning message")

	if event.Type != EventTypeWarning {
		t.Errorf("Warning() Type = %v, want %v", event.Type, EventTypeWarning)
	}

	if event.ResourceKey != "test/key" {
		t.Errorf("Warning() ResourceKey = %v, want test/key", event.ResourceKey)
	}

	if event.Message != "Warning message" {
		t.Errorf("Warning() Message = %v, want Warning message", event.Message)
	}

	if event.Timestamp.IsZero() {
		t.Error("Warning() Timestamp should not be zero")
	}

	if event.Details["operation"] != "update" {
		t.Errorf("Warning() Details[operation] = %v, want update", event.Details["operation"])
	}
}

