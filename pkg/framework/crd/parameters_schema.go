package crd

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetCRDSchema retrieves the OpenAPI schema from the CRD definition using dynamic client
func (c *Client) GetCRDSchema(ctx context.Context) (map[string]interface{}, error) {
	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	
	crdName := fmt.Sprintf("%s.%s", c.resource, c.group)
	crdInterface := c.dynamicClient.Resource(crdGVR)
	
	obj, err := crdInterface.Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get CRD definition %s: %w", crdName, err)
	}

	// Extract the schema from the CRD
	spec, ok := obj.Object["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CRD spec not found")
	}

	versions, ok := spec["versions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("CRD versions not found")
	}

	// Find the version we need
	for _, v := range versions {
		version, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if version["name"] == c.version {
			schema, ok := version["schema"].(map[string]interface{})
			if !ok {
				continue
			}
			openAPIV3Schema, ok := schema["openAPIV3Schema"].(map[string]interface{})
			if ok {
				return openAPIV3Schema, nil
			}
		}
	}

	return nil, fmt.Errorf("schema not found for version %s in CRD %s", c.version, crdName)
}

// GetSpecSchema extracts the spec schema structure from the CRD definition
// Returns a map representing the structure of the spec that can be used for UI rendering
func (c *Client) GetSpecSchema(ctx context.Context) (map[string]interface{}, error) {
	schema, err := c.GetCRDSchema(ctx)
	if err != nil {
		return nil, err
	}

	// Navigate to spec.properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schema properties not found")
	}

	spec, ok := properties["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("spec not found in schema properties")
	}

	specProperties, ok := spec["properties"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("spec properties not found")
	}

	// Build a structure map from the schema
	result := make(map[string]interface{})
	
	// Extract global schema
	if global, ok := specProperties["global"].(map[string]interface{}); ok {
		result["global"] = extractSchemaStructure(global)
	}
	
	// Extract services schema
	if services, ok := specProperties["services"].(map[string]interface{}); ok {
		result["services"] = extractSchemaStructure(services)
	}

	return result, nil
}

// extractSchemaStructure recursively extracts the structure from an OpenAPI schema
func extractSchemaStructure(schema map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// If it has additionalProperties, it's a flexible object
	if additionalProps, ok := schema["additionalProperties"]; ok {
		if additionalProps == true {
			// Allow any properties - return empty map to indicate flexible structure
			return make(map[string]interface{})
		}
		if apMap, ok := additionalProps.(map[string]interface{}); ok {
			// Recursively extract the structure from additionalProperties
			return extractSchemaStructure(apMap)
		}
	}
	
	// Check if it's an array type
	if schemaType, ok := schema["type"].(string); ok && schemaType == "array" {
		// For arrays, extract the items schema if it's an object
		if items, ok := schema["items"].(map[string]interface{}); ok {
			return extractSchemaStructure(items)
		}
		// For arrays of primitives, return nil to indicate it's a primitive array
		return nil
	}
	
	// Extract defined properties
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for key, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				// Check if this is a primitive type (string, number, boolean)
				if propType, ok := propMap["type"].(string); ok {
					if propType == "string" || propType == "number" || propType == "integer" || propType == "boolean" {
						// For primitive types, store the type so template knows it's a leaf node
						result[key] = propType
						continue
					}
				}
				// For complex types, recursively extract
				extracted := extractSchemaStructure(propMap)
				if extracted != nil {
					result[key] = extracted
				} else {
					// Fallback: use empty map
					result[key] = make(map[string]interface{})
				}
			} else {
				result[key] = prop
			}
		}
	} else {
		// If no properties but it's an object type, it might be a primitive object
		// Return empty map to indicate it exists but has no defined structure
		if schemaType, ok := schema["type"].(string); ok && schemaType == "object" {
			return make(map[string]interface{})
		}
	}
	
	return result
}

