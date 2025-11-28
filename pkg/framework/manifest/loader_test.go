package manifest

import (
	"context"
	"embed"
	"fmt"
	"testing"
)

//go:embed testdata/*.yaml
var testManifests embed.FS

func TestExtractKeyFromYAML(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected string
		wantErr  bool
	}{
		{
			name: "deployment with namespace",
			yaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: default
`,
			expected: "default/Deployment/test-deployment",
			wantErr:  false,
		},
		{
			name: "service without namespace",
			yaml: `apiVersion: v1
kind: Service
metadata:
  name: test-service
`,
			expected: "default/Service/test-service",
			wantErr:  false,
		},
		{
			name: "configmap with custom namespace",
			yaml: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: kube-system
`,
			expected: "kube-system/ConfigMap/test-config",
			wantErr:  false,
		},
		{
			name: "missing kind",
			yaml: `apiVersion: v1
metadata:
  name: test
`,
			expected: "",
			wantErr:  true,
		},
		{
			name: "missing name",
			yaml: `apiVersion: v1
kind: Service
metadata: {}
`,
			expected: "",
			wantErr:  true,
		},
		{
			name: "missing metadata",
			yaml: `apiVersion: v1
kind: Service
`,
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := extractKeyFromYAML([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("extractKeyFromYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if key != tt.expected {
				t.Errorf("extractKeyFromYAML() = %v, want %v", key, tt.expected)
			}
		})
	}
}

func TestExtractKeyFromYAMLWithEmptyNamespace(t *testing.T) {
	yaml := `apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: ""
`
	key, err := extractKeyFromYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key != "default/Service/test-service" {
		t.Errorf("expected default/Service/test-service, got %s", key)
	}
}

func TestLoadEmbeddedManifests(t *testing.T) {
	ctx := context.Background()
	manifests, err := LoadEmbeddedManifests(testManifests, "testdata", ctx, nil)
	if err != nil {
		t.Fatalf("LoadEmbeddedManifests() error = %v", err)
	}

	if len(manifests) == 0 {
		t.Error("LoadEmbeddedManifests() should load manifests from testdata")
	}

	for key, yamlData := range manifests {
		if key == "" {
			t.Error("LoadEmbeddedManifests() should not have empty keys")
		}
		if len(yamlData) == 0 {
			t.Errorf("LoadEmbeddedManifests() manifest %s should not be empty", key)
		}
	}
}

func TestLoadEmbeddedManifests_EmptyFS(t *testing.T) {
	ctx := context.Background()
	var emptyFS embed.FS
	manifests, err := LoadEmbeddedManifests(emptyFS, "", ctx, nil)
	if err != nil {
		t.Fatalf("LoadEmbeddedManifests() with empty FS error = %v", err)
	}

	if manifests == nil {
		t.Error("LoadEmbeddedManifests() should return empty map, not nil")
	}
	if len(manifests) != 0 {
		t.Errorf("LoadEmbeddedManifests() with empty FS should return empty map, got %d manifests", len(manifests))
	}
}

func TestLoadEmbeddedManifests_WithRootPath(t *testing.T) {
	ctx := context.Background()
	manifests, err := LoadEmbeddedManifests(testManifests, "testdata", ctx, nil)
	if err != nil {
		t.Fatalf("LoadEmbeddedManifests() with rootPath error = %v", err)
	}

	if len(manifests) == 0 {
		t.Error("LoadEmbeddedManifests() should load manifests with rootPath")
	}
}

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		rootPath string
		expected string
	}{
		{
			name:     "standard path with rootPath",
			path:     "manifests/redis/deployment.yaml",
			rootPath: "manifests",
			expected: "redis",
		},
		{
			name:     "path with leading dot",
			path:     "./manifests/redis/deployment.yaml",
			rootPath: "manifests",
			expected: "redis",
		},
		{
			name:     "path without rootPath prefix",
			path:     "redis/deployment.yaml",
			rootPath: "manifests",
			expected: "redis",
		},
		{
			name:     "empty rootPath",
			path:     "redis/deployment.yaml",
			rootPath: "",
			expected: "redis",
		},
		{
			name:     "nested path",
			path:     "manifests/apps/backend/service.yaml",
			rootPath: "manifests",
			expected: "apps",
		},
		{
			name:     "fallback to default",
			path:     "deployment.yaml",
			rootPath: "",
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.path, tt.rootPath)
			if result != tt.expected {
				t.Errorf("extractServiceName(%q, %q) = %q, want %q", tt.path, tt.rootPath, result, tt.expected)
			}
		})
	}
}

func TestExtractKeyFromYAML_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected string
		wantErr  bool
	}{
		{
			name: "cluster-scoped resource (Namespace)",
			yaml: `apiVersion: v1
kind: Namespace
metadata:
  name: test-namespace
`,
			expected: "default/Namespace/test-namespace",
			wantErr:  false,
		},
		{
			name:     "malformed YAML",
			yaml:     "!!!invalid yaml!!!",
			expected: "",
			wantErr:  true,
		},
		{
			name: "missing apiVersion",
			yaml: `kind: Service
metadata:
  name: test
`,
			expected: "default/Service/test",
			wantErr:  false,
		},
		{
			name: "invalid metadata structure",
			yaml: `apiVersion: v1
kind: Service
metadata: "not a map"
`,
			expected: "",
			wantErr:  true,
		},
		{
			name: "non-string name",
			yaml: `apiVersion: v1
kind: Service
metadata:
  name: 123
`,
			expected: "",
			wantErr:  true,
		},
		{
			name: "non-string kind",
			yaml: `apiVersion: v1
kind: 123
metadata:
  name: test
`,
			expected: "",
			wantErr:  true,
		},
		{
			name: "non-string namespace",
			yaml: `apiVersion: v1
kind: Service
metadata:
  name: test
  namespace: 123
`,
			expected: "default/Service/test",
			wantErr:  false,
		},
		{
			name: "whitespace in name",
			yaml: `apiVersion: v1
kind: Service
metadata:
  name: "test service"
  namespace: default
`,
			expected: "default/Service/test service",
			wantErr:  false,
		},
		{
			name: "very large YAML",
			yaml: func() string {
				yaml := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
`

				for i := 0; i < 1000; i++ {
					yaml += fmt.Sprintf("  key%d: value%d\n", i, i)
				}
				return yaml
			}(),
			expected: "default/ConfigMap/test",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := extractKeyFromYAML([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("extractKeyFromYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if key != tt.expected {
				t.Errorf("extractKeyFromYAML() = %v, want %v", key, tt.expected)
			}
		})
	}
}

