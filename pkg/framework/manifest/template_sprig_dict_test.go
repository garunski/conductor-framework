package manifest

import (
	"strings"
	"testing"
)

func TestRenderTemplate_SprigDictFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{"dict", "{{ (dict \"key1\" \"value1\" \"key2\" \"value2\").key1 }}", "value1"},
		{"get", "{{ get (dict \"key\" \"value\") \"key\" }}", "value"},
		{"hasKey", "{{ hasKey (dict \"key\" \"value\") \"key\" }}", "true"},
		{"keys", "{{ keys (dict \"a\" 1 \"b\" 2) | sortAlpha | join \",\" }}", "a,b"},
		{"merge", "{{ keys (merge (dict \"a\" 1) (dict \"b\" 2)) | sortAlpha | join \",\" }}", "a,b"},
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

