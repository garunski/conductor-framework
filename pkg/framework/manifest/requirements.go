package manifest

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ApplicationRequirement defines a requirement that can be checked by the application
type ApplicationRequirement struct {
	// Name is the display name of the requirement
	Name string `yaml:"name"`
	// Description explains what the requirement is for
	Description string `yaml:"description"`
	// Required indicates if this requirement must pass (true) or is just a warning (false)
	Required bool `yaml:"required"`
	// CheckType specifies the type of check to perform
	// Supported types: "kubernetes-version", "node-count", "storage-class", "cpu", "memory", "custom"
	CheckType string `yaml:"checkType"`
	// CheckConfig contains configuration specific to the check type
	CheckConfig map[string]interface{} `yaml:"checkConfig,omitempty"`
}

// ApplicationRequirementsFile represents the structure of a requirements.yaml file
type ApplicationRequirementsFile struct {
	// Requirements is a list of application-defined requirements
	Requirements []ApplicationRequirement `yaml:"requirements"`
}

// LoadApplicationRequirements loads application requirements from the embedded filesystem
// It looks for requirements.yaml or requirements.yml in the manifests root directory
func LoadApplicationRequirements(files embed.FS, rootPath string) ([]ApplicationRequirement, error) {
	// Default rootPath to "manifests" if empty
	if rootPath == "" {
		rootPath = "manifests"
	}

	// Try requirements.yaml first, then requirements.yml
	var data []byte
	var err error
	var found bool

	for _, filename := range []string{"requirements.yaml", "requirements.yml"} {
		path := filepath.Join(rootPath, filename)
		// Normalize path separators for embed.FS
		path = strings.ReplaceAll(path, "\\", "/")
		
		data, err = files.ReadFile(path)
		if err == nil {
			found = true
			break
		}
		if _, ok := err.(*fs.PathError); !ok {
			// If it's not a path error, return it
			return nil, fmt.Errorf("failed to read requirements file: %w", err)
		}
	}

	if !found {
		// No requirements file found - this is not an error, just return empty list
		return []ApplicationRequirement{}, nil
	}

	var reqFile ApplicationRequirementsFile
	if err := yaml.Unmarshal(data, &reqFile); err != nil {
		return nil, fmt.Errorf("failed to parse requirements file: %w", err)
	}

	return reqFile.Requirements, nil
}

