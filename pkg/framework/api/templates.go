package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"reflect"
	"strings"
)

//go:embed templates/pages/*.html templates/components/*.html templates/partials/*.html templates/static/**/*
var templateFiles embed.FS

// TemplateFiles returns the embedded template filesystem
func TemplateFiles() embed.FS {
	return templateFiles
}

// loadTemplates loads and parses all templates from the embedded filesystem
// If customTemplateFS is provided, it will be used instead of the default embedded templates
func loadTemplates(customTemplateFS *embed.FS) (*template.Template, error) {
	// Use custom templates if provided, otherwise use default embedded templates
	templateFS := templateFiles
	if customTemplateFS != nil {
		templateFS = *customTemplateFS
	}

	// Load templates from all subdirectories
	patterns := []string{
		"templates/pages/*.html",
		"templates/components/*.html",
		"templates/partials/*.html",
	}
	
	// Create template with helper functions
	funcMap := template.FuncMap{
		"isMap": func(v interface{}) bool {
			if v == nil {
				return false
			}
			rv := reflect.ValueOf(v)
			return rv.Kind() == reflect.Map
		},
		"isSlice": func(v interface{}) bool {
			if v == nil {
				return false
			}
			rv := reflect.ValueOf(v)
			return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
		},
		"isBool": func(v interface{}) bool {
			if v == nil {
				return false
			}
			_, ok := v.(bool)
			return ok
		},
		"isString": func(v interface{}) bool {
			if v == nil {
				return false
			}
			_, ok := v.(string)
			return ok
		},
		"isNumber": func(v interface{}) bool {
			if v == nil {
				return false
			}
			rv := reflect.ValueOf(v)
			return rv.Kind() == reflect.Int || rv.Kind() == reflect.Int8 || rv.Kind() == reflect.Int16 ||
				rv.Kind() == reflect.Int32 || rv.Kind() == reflect.Int64 ||
				rv.Kind() == reflect.Uint || rv.Kind() == reflect.Uint8 || rv.Kind() == reflect.Uint16 ||
				rv.Kind() == reflect.Uint32 || rv.Kind() == reflect.Uint64 ||
				rv.Kind() == reflect.Float32 || rv.Kind() == reflect.Float64
		},
		"typeOf": func(v interface{}) string {
			if v == nil {
				return "nil"
			}
			return reflect.TypeOf(v).String()
		},
		"add": func(a, b int) int {
			return a + b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires even number of arguments")
			}
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"default": func(d interface{}, v ...interface{}) interface{} {
			if len(v) == 0 {
				return d
			}
			val := v[0]
			if val == nil {
				return d
			}
			// Check if value is empty string, empty slice, empty map
			rv := reflect.ValueOf(val)
			switch rv.Kind() {
			case reflect.String:
				if rv.Len() == 0 {
					return d
				}
			case reflect.Slice, reflect.Array, reflect.Map:
				if rv.Len() == 0 {
					return d
				}
			}
			return val
		},
		"replace": func(old, new, src string) string {
			return strings.ReplaceAll(src, old, new)
		},
		"title": func(s string) string {
			return strings.Title(strings.ToLower(s))
		},
		"json": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		"safeJS": func(s string) template.JS {
			// JSON is already safe for JavaScript when properly escaped
			return template.JS(s)
		},
	}
	
	tmpl := template.New("").Funcs(funcMap)
	for _, pattern := range patterns {
		// Check if pattern exists in filesystem
		matches, err := fs.Glob(templateFS, pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob templates from %s: %w", pattern, err)
		}
		if len(matches) == 0 {
			// Skip if no matches (allows partial template overrides)
			continue
		}
		
		_, err = tmpl.ParseFS(templateFS, pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to parse templates from %s: %w", pattern, err)
		}
	}
	
	return tmpl, nil
}

// renderTemplate renders a template with the given data and returns the HTML string
func renderTemplate(tmpl *template.Template, name string, data interface{}) (string, error) {
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", name, err)
	}
	return buf.String(), nil
}
