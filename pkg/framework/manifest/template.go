package manifest

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/uuid"
	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

// TemplateContext represents the context passed to Go templates
type TemplateContext struct {
	Namespace    string
	NamePrefix   string
	ServiceName  string
	Replicas     int32
	ImageTag     string
	Resources    *ResourceContext
	StorageSize  string
	Labels       map[string]string
	Annotations  map[string]string
	NodeSelector map[string]string
	Tolerations  []interface{}
}

// ResourceContext represents resource requests and limits in template context
type ResourceContext struct {
	Requests *ResourceListContext
	Limits   *ResourceListContext
}

// ResourceListContext represents memory and CPU in template context
type ResourceListContext struct {
	Memory string
	CPU    string
}

// buildTemplateFuncMap builds a complete function map by merging:
// 1. Existing built-in functions
// 2. Sprig functions (excluding env/expandenv for security)
// 3. Custom uuidv5 function
// 4. User-provided custom functions (highest priority, can override)
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
		"rangeLabels": func() map[string]string {
			return ctx.Labels
		},
		"rangeAnnotations": func() map[string]string {
			return ctx.Annotations
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

// RenderTemplate renders a manifest YAML template with the given parameters
// If customFuncs is provided, it will be merged with built-in and Sprig functions
func RenderTemplate(manifestBytes []byte, serviceName string, params *crd.ParameterSet, customFuncs template.FuncMap) ([]byte, error) {
	if params == nil {
		params = &crd.ParameterSet{
			Namespace:  "default",
			NamePrefix: "",
			Replicas:   int32Ptr(1),
		}
	}

	// Build template context
	ctx := &TemplateContext{
		Namespace:   getStringOrDefault(params.Namespace, "default"),
		NamePrefix:  params.NamePrefix,
		ServiceName: serviceName,
		Replicas:    getReplicasOrDefault(params.Replicas, 1),
		ImageTag:    params.ImageTag,
		StorageSize: params.StorageSize,
	}

	// Set resources
	if params.Resources != nil {
		ctx.Resources = &ResourceContext{}
		if params.Resources.Requests != nil {
			ctx.Resources.Requests = &ResourceListContext{
				Memory: params.Resources.Requests.Memory,
				CPU:    params.Resources.Requests.CPU,
			}
		}
		if params.Resources.Limits != nil {
			ctx.Resources.Limits = &ResourceListContext{
				Memory: params.Resources.Limits.Memory,
				CPU:    params.Resources.Limits.CPU,
			}
		}
	}

	// Set labels and annotations
	if params.Labels != nil {
		ctx.Labels = params.Labels
	} else {
		ctx.Labels = make(map[string]string)
	}

	if params.Annotations != nil {
		ctx.Annotations = params.Annotations
	} else {
		ctx.Annotations = make(map[string]string)
	}

	if params.NodeSelector != nil {
		ctx.NodeSelector = params.NodeSelector
	} else {
		ctx.NodeSelector = make(map[string]string)
	}

	if params.Tolerations != nil {
		ctx.Tolerations = params.Tolerations
	}

	// Build complete function map
	funcMap := buildTemplateFuncMap(ctx, customFuncs)

	// Create template with merged functions
	tmpl, err := template.New("manifest").Funcs(funcMap).Parse(string(manifestBytes))

	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// Helper functions

func int32Ptr(i int32) *int32 {
	return &i
}

func getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func getReplicasOrDefault(value *int32, defaultValue int32) int32 {
	if value == nil {
		return defaultValue
	}
	return *value
}

