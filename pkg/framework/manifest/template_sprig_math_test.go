package manifest

import (
	"context"
	"strings"
	"testing"
)

func TestRenderTemplate_SprigMathFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{"add", "{{ add 1 1 }}", "2"},
		{"sub", "{{ sub 5 2 }}", "3"},
		{"mul", "{{ mul 3 4 }}", "12"},
		{"div", "{{ div 10 2 }}", "5"},
		{"mod", "{{ mod 10 3 }}", "1"},
		{"max", "{{ max 5 10 }}", "10"},
		{"min", "{{ min 5 10 }}", "5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifestBytes := []byte(tt.template)
			spec := make(map[string]interface{})

			result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
			if err != nil {
				t.Fatalf("RenderTemplate() error = %v", err)
			}

			resultStr := strings.TrimSpace(string(result))
			if resultStr != tt.expected {
				t.Errorf("RenderTemplate() = %v, want %v", resultStr, tt.expected)
			}
		})
	}
}

