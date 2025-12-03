package manifest

import (
	"strings"
	"testing"
)

func TestRenderTemplate_SprigListFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{"list", "{{ list \"a\" \"b\" \"c\" | join \",\" }}", "a,b,c"},
		{"first", "{{ first (list \"a\" \"b\" \"c\") }}", "a"},
		{"last", "{{ last (list \"a\" \"b\" \"c\") }}", "c"},
		{"append", "{{ append (list \"a\" \"b\") \"c\" | join \",\" }}", "a,b,c"},
		{"prepend", "{{ prepend (list \"b\" \"c\") \"a\" | join \",\" }}", "a,b,c"},
		{"concat", "{{ concat (list \"a\" \"b\") (list \"c\" \"d\") | join \",\" }}", "a,b,c,d"},
		{"uniq", "{{ uniq (list \"a\" \"b\" \"a\" \"c\") | join \",\" }}", "a,b,c"},
		{"has", "{{ has \"b\" (list \"a\" \"b\" \"c\") }}", "true"},
		{"without", "{{ without (list \"a\" \"b\" \"c\") \"b\" | join \",\" }}", "a,c"},
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

