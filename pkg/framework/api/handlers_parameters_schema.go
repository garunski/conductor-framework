package api

import (
	"net/http"
)

// GetParametersSchema returns the CRD schema for form generation
func (h *Handler) GetParametersSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get CRD schema definition (raw OpenAPI schema) for form generation
	crdSchema, err := h.parameterClient.GetCRDSchema(ctx)
	usingSample := false
	if err != nil {
		h.logger.Info("failed to get CRD schema, using sample schema for local development", "error", err)
		// Use sample schema for local development/debugging
		crdSchema = GetSampleCRDSchema()
		usingSample = true
	}
	
	// Extract the spec schema from the CRD schema
	var specSchema map[string]interface{}
	if properties, ok := crdSchema["properties"].(map[string]interface{}); ok {
		if spec, ok := properties["spec"].(map[string]interface{}); ok {
			specSchema = spec
		}
	}
	
	// If we still don't have a spec schema, use the sample one
	if specSchema == nil || len(specSchema) == 0 {
		h.logger.Info("spec schema not found in CRD, using sample schema")
		sampleSchema := GetSampleCRDSchema()
		if properties, ok := sampleSchema["properties"].(map[string]interface{}); ok {
			if spec, ok := properties["spec"].(map[string]interface{}); ok {
				specSchema = spec
				usingSample = true
			}
		}
	} else {
		// Check if the cluster CRD has descriptions, if not, merge from sample schema
		hasDescriptions := checkSchemaHasDescriptions(specSchema)
		if !hasDescriptions {
			h.logger.Info("cluster CRD missing descriptions, merging from sample schema")
			sampleSchema := GetSampleCRDSchema()
			if properties, ok := sampleSchema["properties"].(map[string]interface{}); ok {
				if sampleSpec, ok := properties["spec"].(map[string]interface{}); ok {
					mergeDescriptions(specSchema, sampleSpec)
				}
			}
		}
	}
	
	if specSchema == nil {
		specSchema = make(map[string]interface{})
	}
	
	// Debug: Check if descriptions are present in the returned schema
	if usingSample {
		h.logger.Info("using sample schema for /api/parameters/schema")
		if specProps, ok := specSchema["properties"].(map[string]interface{}); ok {
			if global, ok := specProps["global"].(map[string]interface{}); ok {
				if globalProps, ok := global["properties"].(map[string]interface{}); ok {
					if namespace, ok := globalProps["namespace"].(map[string]interface{}); ok {
						if desc, ok := namespace["description"].(string); ok {
							h.logger.Info("sample schema has description for namespace field", "description", desc)
						} else {
							h.logger.Info("sample schema missing description for namespace field", "namespace", namespace)
						}
					} else {
						h.logger.Info("namespace field not found in global properties")
					}
				} else {
					h.logger.Info("global properties not found")
				}
			} else {
				h.logger.Info("global not found in spec properties")
			}
		} else {
			h.logger.Info("spec properties not found in specSchema")
		}
	} else {
		h.logger.Info("using CRD schema from cluster (not sample schema)")
	}
	
	WriteJSONResponse(w, h.logger, http.StatusOK, specSchema)
}

// checkSchemaHasDescriptions checks if a schema has any descriptions
func checkSchemaHasDescriptions(schema map[string]interface{}) bool {
	if schema == nil {
		return false
	}
	
	// Check if this level has a description
	if _, ok := schema["description"].(string); ok {
		return true
	}
	
	// Recursively check properties
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for _, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if checkSchemaHasDescriptions(propMap) {
					return true
				}
			}
		}
	}
	
	// Check items for arrays
	if items, ok := schema["items"].(map[string]interface{}); ok {
		return checkSchemaHasDescriptions(items)
	}
	
	return false
}

// mergeDescriptions merges descriptions from sample schema into cluster schema
func mergeDescriptions(clusterSchema, sampleSchema map[string]interface{}) {
	if clusterSchema == nil || sampleSchema == nil {
		return
	}
	
	// Merge description at this level
	if sampleDesc, ok := sampleSchema["description"].(string); ok {
		if _, exists := clusterSchema["description"]; !exists {
			clusterSchema["description"] = sampleDesc
		}
	}
	
	// Recursively merge properties
	clusterProps, clusterOk := clusterSchema["properties"].(map[string]interface{})
	sampleProps, sampleOk := sampleSchema["properties"].(map[string]interface{})
	
	if clusterOk && sampleOk {
		for key, sampleProp := range sampleProps {
			if samplePropMap, ok := sampleProp.(map[string]interface{}); ok {
				if clusterProp, exists := clusterProps[key]; exists {
					if clusterPropMap, ok := clusterProp.(map[string]interface{}); ok {
						mergeDescriptions(clusterPropMap, samplePropMap)
					}
				}
			}
		}
	}
	
	// Merge items for arrays
	clusterItems, clusterItemsOk := clusterSchema["items"].(map[string]interface{})
	sampleItems, sampleItemsOk := sampleSchema["items"].(map[string]interface{})
	if clusterItemsOk && sampleItemsOk {
		mergeDescriptions(clusterItems, sampleItems)
	}
}

