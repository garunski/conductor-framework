package manifest

import (
	"strings"
	"testing"
)

func TestRenderTemplate_SprigEncodingFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{"b64enc", "{{ b64enc \"hello\" }}", "aGVsbG8="},
		{"b64dec", "{{ b64dec \"aGVsbG8=\" }}", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifestBytes := []byte(tt.template)
			spec := make(map[string]interface{})

			result, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
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

