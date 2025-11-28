package api

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
)

//go:embed templates/pages/*.html templates/components/*.html templates/partials/*.html
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
	
	tmpl := template.New("")
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
