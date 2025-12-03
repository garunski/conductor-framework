package manifest

import (
	"context"
	"strings"
	"testing"
)

func TestRenderTemplate_UUIDv5_Deterministic(t *testing.T) {
	manifestBytes := []byte("{{ uuidv5 \"6ba7b810-9dad-11d1-80b4-00c04fd430c8\" \"test-name\" }}")
	spec := make(map[string]interface{})

	// First call
	result1, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	// Second call with same inputs
	result2, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	result1Str := strings.TrimSpace(string(result1))
	result2Str := strings.TrimSpace(string(result2))

	if result1Str != result2Str {
		t.Errorf("UUIDv5 should be deterministic: got %v and %v", result1Str, result2Str)
	}

	// Should be a valid UUID format
	if len(result1Str) != 36 {
		t.Errorf("UUID should be 36 characters, got %d", len(result1Str))
	}
}

func TestRenderTemplate_UUIDv5_DifferentInputs(t *testing.T) {
	spec := make(map[string]interface{})

	// Different names should produce different UUIDs
	manifest1 := []byte("{{ uuidv5 \"6ba7b810-9dad-11d1-80b4-00c04fd430c8\" \"name1\" }}")
	manifest2 := []byte("{{ uuidv5 \"6ba7b810-9dad-11d1-80b4-00c04fd430c8\" \"name2\" }}")

	result1, err := RenderTemplate(context.Background(), manifest1, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	result2, err := RenderTemplate(context.Background(), manifest2, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	result1Str := strings.TrimSpace(string(result1))
	result2Str := strings.TrimSpace(string(result2))

	if result1Str == result2Str {
		t.Errorf("Different inputs should produce different UUIDs: got %v for both", result1Str)
	}
}

func TestRenderTemplate_UUIDv5_InvalidNamespace(t *testing.T) {
	manifestBytes := []byte("{{ uuidv5 \"invalid-uuid\" \"test-name\" }}")
	spec := make(map[string]interface{})

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() should not error on invalid namespace UUID: %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	// Should return empty string on error
	if resultStr != "" {
		t.Errorf("Invalid namespace UUID should return empty string, got %v", resultStr)
	}
}

func TestRenderTemplate_UUIDv5_EmptyNamespace(t *testing.T) {
	manifestBytes := []byte("{{ uuidv5 \"\" \"test-name\" }}")
	spec := make(map[string]interface{})

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	// Should use DNS namespace UUID as default
	if resultStr == "" {
		t.Errorf("Empty namespace should use default DNS namespace UUID, got empty string")
	}
	if len(resultStr) != 36 {
		t.Errorf("UUID should be 36 characters, got %d", len(resultStr))
	}
}

