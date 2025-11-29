package manifest

import (
	"strings"
	"testing"
	"text/template"
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

func TestRenderTemplate_UUIDv5_Deterministic(t *testing.T) {
	manifestBytes := []byte("{{ uuidv5 \"6ba7b810-9dad-11d1-80b4-00c04fd430c8\" \"test-name\" }}")
	spec := make(map[string]interface{})

	// First call
	result1, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	// Second call with same inputs
	result2, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
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

	result1, err := RenderTemplate(manifest1, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	result2, err := RenderTemplate(manifest2, "test", spec, nil, nil)
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

	result, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
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

	result, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
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

func TestRenderTemplate_CustomFunctionOverride(t *testing.T) {
	// Create a custom function that overrides Sprig's upper function
	customFuncs := template.FuncMap{
		"upper": func(s string) string {
			return "OVERRIDDEN-" + s
		},
	}

	manifestBytes := []byte("{{ upper \"test\" }}")
	spec := make(map[string]interface{})

	result, err := RenderTemplate(manifestBytes, "test", spec, nil, customFuncs)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	expected := "OVERRIDDEN-test"
	if resultStr != expected {
		t.Errorf("Custom function should override Sprig function: got %v, want %v", resultStr, expected)
	}
}

func TestRenderTemplate_CustomFunctionAvailable(t *testing.T) {
	// Create a custom function
	customFuncs := template.FuncMap{
		"customFunc": func(s string) string {
			return "custom-" + s
		},
	}

	manifestBytes := []byte("{{ customFunc \"test\" }}")
	spec := make(map[string]interface{})

	result, err := RenderTemplate(manifestBytes, "test", spec, nil, customFuncs)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	expected := "custom-test"
	if resultStr != expected {
		t.Errorf("Custom function should be available: got %v, want %v", resultStr, expected)
	}
}

func TestRenderTemplate_ExistingFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{"defaultIfEmpty", "{{ defaultIfEmpty \"\" \"default\" }}", "default"},
		{"prefixName", "{{ prefixName \"pre-\" \"name\" }}", "pre-name"},
		{"hasPrefix", "{{ hasPrefix \"pre-\" \"pre-name\" }}", "true"},
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

func TestRenderTemplate_NilFunctionMap(t *testing.T) {
	manifestBytes := []byte("{{ upper \"test\" }}")
	spec := make(map[string]interface{})

	// Should work with nil function map (uses defaults)
	result, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() should work with nil function map: %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	if resultStr != "TEST" {
		t.Errorf("Nil function map should use default functions: got %v, want TEST", resultStr)
	}
}

func TestRenderTemplate_EnvFunctionsExcluded(t *testing.T) {
	manifestBytes := []byte("{{ env \"TEST_VAR\" }}")
	spec := make(map[string]interface{})

	// Should error because env function is excluded
	_, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
	if err == nil {
		t.Error("env function should be excluded and cause an error")
	}
}

func TestRenderTemplate_ExpandenvFunctionsExcluded(t *testing.T) {
	manifestBytes := []byte("{{ expandenv \"$TEST_VAR\" }}")
	spec := make(map[string]interface{})

	// Should error because expandenv function is excluded
	_, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
	if err == nil {
		t.Error("expandenv function should be excluded and cause an error")
	}
}

// New tests for the new functionality

func TestRenderTemplate_SpecGlobalAccess(t *testing.T) {
	manifestBytes := []byte("namespace: {{ .Spec.global.namespace }}")
	spec := map[string]interface{}{
		"global": map[string]interface{}{
			"namespace": "test-namespace",
		},
	}

	result, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	expected := "namespace: test-namespace"
	if resultStr != expected {
		t.Errorf("RenderTemplate() = %v, want %v", resultStr, expected)
	}
}

func TestRenderTemplate_SpecServicesAccess(t *testing.T) {
	manifestBytes := []byte("replicas: {{ .Spec.services.frontend.replicas }}")
	spec := map[string]interface{}{
		"services": map[string]interface{}{
			"frontend": map[string]interface{}{
				"replicas": 3,
			},
		},
	}

	result, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	expected := "replicas: 3"
	if resultStr != expected {
		t.Errorf("RenderTemplate() = %v, want %v", resultStr, expected)
	}
}

func TestRenderTemplate_NestedConfigAccess(t *testing.T) {
	// Use getService helper for hyphenated service names
	manifestBytes := []byte("{{- $svc := getService \"api-service\" }}host: {{ $svc.config.database.host }}")
	spec := map[string]interface{}{
		"services": map[string]interface{}{
			"api-service": map[string]interface{}{
				"config": map[string]interface{}{
					"database": map[string]interface{}{
						"host": "redis-master",
					},
				},
			},
		},
	}

	result, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	expected := "host: redis-master"
	if resultStr != expected {
		t.Errorf("RenderTemplate() = %v, want %v", resultStr, expected)
	}
}

func TestRenderTemplate_GetServiceHelper(t *testing.T) {
	manifestBytes := []byte("{{- $svc := getService \"otel-collector\" }}{{ if $svc }}found{{ else }}not found{{ end }}")
	spec := map[string]interface{}{
		"services": map[string]interface{}{
			"otel-collector": map[string]interface{}{
				"port": 4317,
			},
		},
	}

	result, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	expected := "found"
	if resultStr != expected {
		t.Errorf("RenderTemplate() = %v, want %v", resultStr, expected)
	}
}

func TestRenderTemplate_GetServiceHelperNotFound(t *testing.T) {
	manifestBytes := []byte("{{- $svc := getService \"nonexistent\" }}{{ if $svc }}found{{ else }}not found{{ end }}")
	spec := map[string]interface{}{
		"services": map[string]interface{}{
			"other-service": map[string]interface{}{},
		},
	}

	result, err := RenderTemplate(manifestBytes, "test", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	expected := "not found"
	if resultStr != expected {
		t.Errorf("RenderTemplate() = %v, want %v", resultStr, expected)
	}
}

func TestRenderTemplate_FilesGet(t *testing.T) {
	// Create a test filesystem
	testFS := &FileSystem{
		fs:       testManifests,
		rootPath: "testdata",
	}

	manifestBytes := []byte("{{ .Files.Get \"test.yaml\" }}")
	spec := make(map[string]interface{})

	result, err := RenderTemplate(manifestBytes, "test", spec, testFS, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := string(result)
	// Should contain content from test.yaml file
	if !strings.Contains(resultStr, "apiVersion") {
		t.Errorf("Files.Get() should return file content, got: %v", resultStr)
	}
}

func TestRenderTemplate_FilesGetNotFound(t *testing.T) {
	testFS := &FileSystem{
		fs:       testManifests,
		rootPath: "testdata",
	}

	manifestBytes := []byte("{{ .Files.Get \"nonexistent.yaml\" }}")
	spec := make(map[string]interface{})

	result, err := RenderTemplate(manifestBytes, "test", spec, testFS, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	// Should return empty string for non-existent file
	if resultStr != "" {
		t.Errorf("Files.Get() should return empty string for non-existent file, got: %v", resultStr)
	}
}

func TestRenderTemplate_CrossServiceReference(t *testing.T) {
	// Use getService helper for hyphenated service names
	manifestBytes := []byte("{{- $redis := getService \"redis-master\" }}redis-port: {{ $redis.port }}")
	spec := map[string]interface{}{
		"services": map[string]interface{}{
			"redis-master": map[string]interface{}{
				"port": 6379,
			},
			"frontend": map[string]interface{}{
				"replicas": 3,
			},
		},
	}

	result, err := RenderTemplate(manifestBytes, "frontend", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	expected := "redis-port: 6379"
	if resultStr != expected {
		t.Errorf("RenderTemplate() = %v, want %v", resultStr, expected)
	}
}
