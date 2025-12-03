package manifest

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/uuid"
)

// TemplateContext represents the context passed to Go templates
type TemplateContext struct {
	Spec  map[string]interface{} // Full CRD spec: .Spec.Global, .Spec.Services
	Files *FileSystem             // For .Files.Get() support
}

// FileSystem provides access to embedded files for templates
type FileSystem struct {
	fs       embed.FS
	rootPath string
}

// Get reads a file from the embedded filesystem relative to rootPath
func (fs *FileSystem) Get(path string) string {
	if fs == nil {
		return ""
	}
	
	// Join rootPath with the requested path
	fullPath := filepath.Join(fs.rootPath, path)
	
	data, err := fs.fs.ReadFile(fullPath)
	if err != nil {
		return ""
	}
	
	return string(data)
}

// buildTemplateFuncMap builds a complete function map by merging:
// 1. Existing built-in functions
// 2. Sprig functions (excluding env/expandenv for security)
// 3. Custom uuidv5 function
// 4. getService helper for hyphenated service names
// 5. User-provided custom functions (highest priority, can override)
func buildTemplateFuncMap(ctx *TemplateContext, customFuncs template.FuncMap) template.FuncMap {
	// Start with existing built-in functions
	funcMap := template.FuncMap{
		"defaultIfEmpty": func(value, defaultValue string) string {
			if value == "" {
				return defaultValue
			}
			return value
		},
		"prefixName": func(prefix, name string) string {
			if prefix == "" {
				return name
			}
			return prefix + name
		},
		"hasPrefix": func(prefix string) bool {
			return prefix != ""
		},
		"toYAML": func(v interface{}) string {
			// Simple YAML conversion for maps
			if m, ok := v.(map[string]string); ok {
				var parts []string
				for k, val := range m {
					parts = append(parts, fmt.Sprintf("%s: %s", k, val))
				}
				return strings.Join(parts, "\n")
			}
			return ""
		},
		// getService helper for accessing services with hyphenated names
		"getService": func(serviceName string) interface{} {
			if ctx == nil || ctx.Spec == nil {
				return nil
			}
			services, ok := ctx.Spec["services"].(map[string]interface{})
			if !ok {
				return nil
			}
			return services[serviceName]
		},
	}

	// Add Sprig functions, but exclude env and expandenv for security
	sprigFuncs := sprig.FuncMap()
	delete(sprigFuncs, "env")
	delete(sprigFuncs, "expandenv")
	
	// Merge Sprig functions
	for k, v := range sprigFuncs {
		funcMap[k] = v
	}

	// Add custom uuidv5 function
	funcMap["uuidv5"] = func(namespaceUUID, name string) string {
		// Default to DNS namespace UUID if empty
		if namespaceUUID == "" {
			namespaceUUID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
		}
		
		nsUUID, err := uuid.Parse(namespaceUUID)
		if err != nil {
			// Return empty string on error (could also log, but template functions shouldn't error)
			return ""
		}
		
		newUUID := uuid.NewSHA1(nsUUID, []byte(name))
		return newUUID.String()
	}

	// Merge user-provided custom functions (highest priority, can override)
	if customFuncs != nil {
		for k, v := range customFuncs {
			funcMap[k] = v
		}
	}

	return funcMap
}

// RenderTemplate renders a manifest YAML template with the given spec and filesystem
// If customFuncs is provided, it will be merged with built-in and Sprig functions
// Context is used for cancellation and timeout handling during template rendering
func RenderTemplate(ctx context.Context, manifestBytes []byte, serviceName string, spec map[string]interface{}, files *FileSystem, customFuncs template.FuncMap) ([]byte, error) {
	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Ensure spec is not nil
	if spec == nil {
		spec = make(map[string]interface{})
	}

	// Build template context
	templateCtx := &TemplateContext{
		Spec:  spec,
		Files: files,
	}

	// Build complete function map
	funcMap := buildTemplateFuncMap(templateCtx, customFuncs)

	// Create template with merged functions
	tmpl, err := template.New("manifest").Funcs(funcMap).Parse(string(manifestBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Check for context cancellation before execution
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateCtx); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

