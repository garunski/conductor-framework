package api

import (
	"strings"
	"testing"
)

func TestIsValidKubernetesName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid name", "test-service", true},
		{"valid with numbers", "test-123", true},
		{"valid single char", "a", true},
		{"empty string", "", false},
		{"starts with dash", "-test", false},
		{"ends with dash", "test-", false},
		{"uppercase", "Test", false},
		{"underscore", "test_service", false},
		{"too long", strings.Repeat("a", 254), false},
		{"valid max length", strings.Repeat("a", 253), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidKubernetesName(tt.input)
			if got != tt.expected {
				t.Errorf("isValidKubernetesName(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCheckSchemaHasDescriptions(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected bool
	}{
		{
			name: "has description at root",
			schema: map[string]interface{}{
				"description": "root description",
			},
			expected: true,
		},
		{
			name: "has description in properties",
			schema: map[string]interface{}{
				"properties": map[string]interface{}{
					"field": map[string]interface{}{
						"description": "field description",
					},
				},
			},
			expected: true,
		},
		{
			name: "has description in nested properties",
			schema: map[string]interface{}{
				"properties": map[string]interface{}{
					"nested": map[string]interface{}{
						"properties": map[string]interface{}{
							"field": map[string]interface{}{
								"description": "nested description",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "has description in items",
			schema: map[string]interface{}{
				"items": map[string]interface{}{
					"description": "item description",
				},
			},
			expected: true,
		},
		{
			name: "no descriptions",
			schema: map[string]interface{}{
				"properties": map[string]interface{}{
					"field": map[string]interface{}{
						"type": "string",
					},
				},
			},
			expected: false,
		},
		{
			name:     "nil schema",
			schema:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkSchemaHasDescriptions(tt.schema)
			if got != tt.expected {
				t.Errorf("checkSchemaHasDescriptions() = %v, want %v", got, tt.expected)
			}
		})
	}
}

