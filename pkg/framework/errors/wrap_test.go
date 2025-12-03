package errors

import (
	"errors"
	"testing"
)

func TestWrapKubernetes(t *testing.T) {
	originalErr := errors.New("connection failed")
	wrapped := WrapKubernetes(originalErr, "failed to get pod")

	if wrapped == nil {
		t.Fatal("WrapKubernetes() should not return nil")
	}

	if !errors.Is(wrapped, ErrKubernetes) {
		t.Error("WrapKubernetes() should wrap with ErrKubernetes")
	}

	if !errors.Is(wrapped, originalErr) {
		t.Error("WrapKubernetes() should preserve original error")
	}

	// Test nil error
	if WrapKubernetes(nil, "context") != nil {
		t.Error("WrapKubernetes() should return nil for nil error")
	}
}

func TestWrapStorage(t *testing.T) {
	originalErr := errors.New("write failed")
	wrapped := WrapStorage(originalErr, "failed to store event")

	if wrapped == nil {
		t.Fatal("WrapStorage() should not return nil")
	}

	if !errors.Is(wrapped, ErrStorage) {
		t.Error("WrapStorage() should wrap with ErrStorage")
	}

	if !errors.Is(wrapped, originalErr) {
		t.Error("WrapStorage() should preserve original error")
	}

	// Test nil error
	if WrapStorage(nil, "context") != nil {
		t.Error("WrapStorage() should return nil for nil error")
	}
}

func TestWrapInvalid(t *testing.T) {
	originalErr := errors.New("invalid input")
	wrapped := WrapInvalid(originalErr, "invalid service name")

	if wrapped == nil {
		t.Fatal("WrapInvalid() should not return nil")
	}

	if !errors.Is(wrapped, ErrInvalid) {
		t.Error("WrapInvalid() should wrap with ErrInvalid")
	}

	if !errors.Is(wrapped, originalErr) {
		t.Error("WrapInvalid() should preserve original error")
	}

	// Test nil error
	if WrapInvalid(nil, "context") != nil {
		t.Error("WrapInvalid() should return nil for nil error")
	}
}

func TestWrapNotFound(t *testing.T) {
	originalErr := errors.New("not found")
	wrapped := WrapNotFound(originalErr, "service not found")

	if wrapped == nil {
		t.Fatal("WrapNotFound() should not return nil")
	}

	if !errors.Is(wrapped, ErrNotFound) {
		t.Error("WrapNotFound() should wrap with ErrNotFound")
	}

	if !errors.Is(wrapped, originalErr) {
		t.Error("WrapNotFound() should preserve original error")
	}

	// Test nil error
	if WrapNotFound(nil, "context") != nil {
		t.Error("WrapNotFound() should return nil for nil error")
	}
}

func TestWrapInvalidYAML(t *testing.T) {
	originalErr := errors.New("parse error")
	wrapped := WrapInvalidYAML(originalErr, "failed to parse manifest")

	if wrapped == nil {
		t.Fatal("WrapInvalidYAML() should not return nil")
	}

	if !errors.Is(wrapped, ErrInvalidYAML) {
		t.Error("WrapInvalidYAML() should wrap with ErrInvalidYAML")
	}

	if !errors.Is(wrapped, originalErr) {
		t.Error("WrapInvalidYAML() should preserve original error")
	}

	// Test nil error
	if WrapInvalidYAML(nil, "context") != nil {
		t.Error("WrapInvalidYAML() should return nil for nil error")
	}
}

