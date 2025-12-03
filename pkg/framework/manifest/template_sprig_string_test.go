package manifest

import (
	"strings"
	"testing"
)

func TestRenderTemplate_SprigStringFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{"upper", "{{ upper \"redis\" }}", "REDIS"},
		{"lower", "{{ lower \"REDIS\" }}", "redis"},
		{"title", "{{ title \"redis\" }}", "Redis"},
		{"trim", "{{ trim \"  test  \" }}", "test"},
		{"trimPrefix", "{{ trimPrefix \"redis-\" \"redis-master\" }}", "master"},
		{"trimSuffix", "{{ trimSuffix \"-service\" \"my-service\" }}", "my"},
		{"replace", "{{ replace \"old\" \"new\" \"old value\" }}", "new value"},
		{"contains", "{{ contains \"test\" \"testing\" }}", "true"},
		{"hasPrefix", "{{ hasPrefix \"test\" \"testing\" }}", "true"},
		{"hasSuffix", "{{ hasSuffix \"ing\" \"testing\" }}", "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifestBytes := []byte(tt.template)
			spec := make(map[string]interface{})

			result, err := RenderTemplate(manifestBytes, "redis", spec, nil, nil)
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

