package manifest

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

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

// RenderTemplate renders a manifest YAML template with the given parameters
func RenderTemplate(manifestBytes []byte, serviceName string, params *crd.ParameterSet) ([]byte, error) {
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

	// Create template with helper functions
	tmpl, err := template.New("manifest").Funcs(template.FuncMap{
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
	}).Parse(string(manifestBytes))

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

