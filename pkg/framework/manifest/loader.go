package manifest

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// ParameterGetter is a function type that retrieves the full CRD spec
type ParameterGetter func(ctx context.Context) (map[string]interface{}, error)

// LoadEmbeddedManifests loads and optionally renders templates for embedded manifests
// If parameterGetter is nil, manifests are loaded without templating
// If templateFuncs is nil, default functions (Sprig + built-ins) are used
// rootPath specifies the root directory path in the embedded filesystem (e.g., "manifests" or "")
func LoadEmbeddedManifests(files embed.FS, rootPath string, ctx context.Context, parameterGetter ParameterGetter, templateFuncs template.FuncMap) (map[string][]byte, error) {
	manifests := make(map[string][]byte)

	// Default rootPath to "manifests" if empty for backward compatibility
	if rootPath == "" {
		rootPath = "manifests"
	}

	// Check if rootPath exists in the filesystem
	_, err := fs.Stat(files, rootPath)
	if err != nil {
		// If rootPath doesn't exist, return empty map (no manifests to load)
		return manifests, nil
	}

	// Get full spec once at the start (not per-service)
	var spec map[string]interface{}
	if parameterGetter != nil {
		var err error
		spec, err = parameterGetter(ctx)
		if err != nil {
			// If parameter getter fails (e.g., no Kubernetes connection),
			// fall back to empty spec which will use defaults in templates
			spec = make(map[string]interface{})
		}
	} else {
		spec = make(map[string]interface{})
	}

	// Create FileSystem instance for .Files.Get() support
	fileSystem := &FileSystem{
		fs:       files,
		rootPath: rootPath,
	}

	err = fs.WalkDir(files, rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// Skip requirements.yaml and requirements.yml files - these are not Kubernetes manifests
		baseName := filepath.Base(path)
		if baseName == "requirements.yaml" || baseName == "requirements.yml" {
			return nil
		}

		data, err := files.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Extract service name from path using rootPath (for potential future use or logging)
		serviceName := extractServiceName(path, rootPath)

		// Render template with full spec and filesystem
		rendered, err := RenderTemplate(ctx, data, serviceName, spec, fileSystem, templateFuncs)
		if err != nil {
			return fmt.Errorf("failed to render template for %s: %w", path, err)
		}
		data = rendered

		// Skip empty or whitespace-only rendered templates (e.g., conditional manifests that are disabled)
		if strings.TrimSpace(string(data)) == "" {
			return nil
		}

		key, err := extractKeyFromYAML(data)
		if err != nil {
			return fmt.Errorf("failed to extract key from %s: %w (file may be missing required Kubernetes fields)", path, err)
		}

		manifests[key] = data
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk embedded manifests: %w", err)
	}

	return manifests, nil
}

// extractServiceName extracts the service name from a manifest file path
// e.g., "manifests/redis/deployment.yaml" with rootPath "manifests" -> "redis"
func extractServiceName(path string, rootPath string) string {
	// Remove leading rootPath if present
	path = strings.TrimPrefix(path, rootPath+"/")
	path = strings.TrimPrefix(path, "./"+rootPath+"/")
	path = strings.TrimPrefix(path, rootPath)
	path = strings.TrimPrefix(path, "./")

	// Get the first directory component
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 0 && parts[0] != "" {
		// If the first part is a filename (has extension), return default
		if strings.Contains(parts[0], ".") {
			return "default"
		}
		return parts[0]
	}

	// Fallback: try to extract from path
	dir := filepath.Dir(path)
	if dir != "." && dir != "" && dir != rootPath {
		return filepath.Base(dir)
	}

	return "default"
}

func extractKeyFromYAML(yamlData []byte) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &obj); err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	metadata, ok := obj["metadata"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing metadata in YAML")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return "", fmt.Errorf("missing metadata.name in YAML")
	}

	namespace := "default"
	if ns, ok := metadata["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	kind, ok := obj["kind"].(string)
	if !ok {
		return "", fmt.Errorf("missing kind in YAML")
	}

	return fmt.Sprintf("%s/%s/%s", namespace, kind, name), nil
}

