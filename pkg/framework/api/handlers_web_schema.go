package api

import (
	"embed"

	"gopkg.in/yaml.v3"
)

//go:embed sample-crd-schema.yaml
var sampleSchemaData embed.FS

// GetSampleCRDSchema returns a sample CRD schema for local development/debugging
// This is used when the real CRD cannot be fetched from the cluster
// The schema is loaded from an embedded YAML file
func GetSampleCRDSchema() map[string]interface{} {
	data, err := sampleSchemaData.ReadFile("sample-crd-schema.yaml")
	if err != nil {
		// Fallback to empty schema if file cannot be read
		// Note: This error will be silent, but the caller should handle empty schema
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	var schema map[string]interface{}
	if err := yaml.Unmarshal(data, &schema); err != nil {
		// Fallback to empty schema if YAML cannot be parsed
		// Note: This error will be silent, but the caller should handle empty schema
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	// Verify the schema structure
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		if spec, ok := properties["spec"].(map[string]interface{}); ok {
			if specProps, ok := spec["properties"].(map[string]interface{}); ok {
				if global, ok := specProps["global"].(map[string]interface{}); ok {
					if globalProps, ok := global["properties"].(map[string]interface{}); ok {
						if namespace, ok := globalProps["namespace"].(map[string]interface{}); ok {
							if _, hasDesc := namespace["description"]; !hasDesc {
								// Schema loaded but missing descriptions - this shouldn't happen
							}
						}
					}
				}
			}
		}
	}

	return schema
}

// mergeSchemaWithInstance merges the CRD schema structure with instance values
// Schema provides the structure (what fields exist), instance provides the values
func mergeSchemaWithInstance(schema, instance map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Start with schema structure
	if schemaGlobal, ok := schema["global"].(map[string]interface{}); ok {
		result["global"] = deepCopyMap(schemaGlobal)
	} else {
		result["global"] = make(map[string]interface{})
	}
	
	// Get schema template for services (this is a template for what each service can have)
	var schemaServicesTemplate map[string]interface{}
	if schemaServices, ok := schema["services"].(map[string]interface{}); ok {
		schemaServicesTemplate = schemaServices
	}
	
	// Initialize services map
	result["services"] = make(map[string]interface{})
	
	// Overlay instance values on top of schema structure
	if instanceGlobal, ok := instance["global"].(map[string]interface{}); ok {
		mergeMaps(result["global"].(map[string]interface{}), instanceGlobal)
	}
	
	// For services, the schema structure is a template for each service
	// Merge schema template with each service's instance values
	if instanceServices, ok := instance["services"].(map[string]interface{}); ok {
		for serviceName, serviceInstance := range instanceServices {
			if serviceMap, ok := serviceInstance.(map[string]interface{}); ok {
				// Start with schema template if available
				var serviceResult map[string]interface{}
				if schemaServicesTemplate != nil {
					serviceResult = deepCopyMap(schemaServicesTemplate)
				} else {
					serviceResult = make(map[string]interface{})
				}
				// Merge instance values into the schema template
				mergeMaps(serviceResult, serviceMap)
				// Store the merged result
				result["services"].(map[string]interface{})[serviceName] = serviceResult
			} else {
				// If not a map, just use the instance value
				result["services"].(map[string]interface{})[serviceName] = serviceInstance
			}
		}
	}
	
	return result
}

// deepCopyMap creates a deep copy of a map
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		if vMap, ok := v.(map[string]interface{}); ok {
			result[k] = deepCopyMap(vMap)
		} else {
			result[k] = v
		}
	}
	return result
}

// mergeMaps merges source into destination, recursively
func mergeMaps(dest, src map[string]interface{}) {
	for k, v := range src {
		if vMap, ok := v.(map[string]interface{}); ok {
			if destMap, ok := dest[k].(map[string]interface{}); ok {
				mergeMaps(destMap, vMap)
			} else {
				dest[k] = deepCopyMap(vMap)
			}
		} else {
			dest[k] = v
		}
	}
}

