package manifest

import (
	"context"
	"strings"
	"testing"
	"text/template"
)

func TestRenderTemplate_CustomFunctionOverride(t *testing.T) {
	// Create a custom function that overrides Sprig's upper function
	customFuncs := template.FuncMap{
		"upper": func(s string) string {
			return "OVERRIDDEN-" + s
		},
	}

	manifestBytes := []byte("{{ upper \"test\" }}")
	spec := make(map[string]interface{})

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, customFuncs)
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

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, customFuncs)
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

func TestRenderTemplate_NilFunctionMap(t *testing.T) {
	manifestBytes := []byte("{{ upper \"test\" }}")
	spec := make(map[string]interface{})

	// Should work with nil function map (uses defaults)
	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
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
	_, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
	if err == nil {
		t.Error("env function should be excluded and cause an error")
	}
}

func TestRenderTemplate_ExpandenvFunctionsExcluded(t *testing.T) {
	manifestBytes := []byte("{{ expandenv \"$TEST_VAR\" }}")
	spec := make(map[string]interface{})

	// Should error because expandenv function is excluded
	_, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
	if err == nil {
		t.Error("expandenv function should be excluded and cause an error")
	}
}

func TestRenderTemplate_SpecGlobalAccess(t *testing.T) {
	manifestBytes := []byte("namespace: {{ .Spec.global.namespace }}")
	spec := map[string]interface{}{
		"global": map[string]interface{}{
			"namespace": "test-namespace",
		},
	}

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
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

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
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

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
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

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
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

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, nil, nil)
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

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, testFS, nil)
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

	result, err := RenderTemplate(context.Background(), manifestBytes, "test", spec, testFS, nil)
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

	result, err := RenderTemplate(context.Background(), manifestBytes, "frontend", spec, nil, nil)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	resultStr := strings.TrimSpace(string(result))
	expected := "redis-port: 6379"
	if resultStr != expected {
		t.Errorf("RenderTemplate() = %v, want %v", resultStr, expected)
	}
}

