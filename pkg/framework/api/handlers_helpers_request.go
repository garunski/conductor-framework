package api

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
)

// getNamespaceAndInstance extracts the namespace and instance name from the request.
// It returns the detected namespace (from manifests) and instance name (from query parameter).
// The namespace defaults to "default" if not detected.
func (h *Handler) getNamespaceAndInstance(r *http.Request) (namespace, instance string) {
	instance = getInstanceName(r)
	namespace = h.getDetectedNamespace()
	if namespace == "" {
		namespace = "default"
	}
	return namespace, instance
}

// getCRDSchemaWithFallback retrieves the CRD schema with fallback to sample schema.
// It extracts the spec schema from the CRD schema properties.
// Returns the spec schema map, or nil if extraction fails.
func (h *Handler) getCRDSchemaWithFallback(ctx context.Context) (map[string]interface{}, error) {
	crdSchema, err := h.parameterClient.GetCRDSchema(ctx)
	if err != nil {
		h.logger.V(1).Info("failed to get CRD schema, using sample schema", "error", err)
		crdSchema = GetSampleCRDSchema()
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
		sampleSchema := GetSampleCRDSchema()
		if properties, ok := sampleSchema["properties"].(map[string]interface{}); ok {
			if spec, ok := properties["spec"].(map[string]interface{}); ok {
				specSchema = spec
			}
		}
	}

	return specSchema, nil
}

// getSpecWithFallback retrieves the parameter spec with fallback to default namespace.
// It tries the detected namespace first, then falls back to "default" namespace if not found.
func (h *Handler) getSpecWithFallback(ctx context.Context, instanceName, detectedNamespace string) (map[string]interface{}, error) {
	spec, err := h.parameterClient.GetSpec(ctx, instanceName, detectedNamespace)
	if err != nil || spec == nil || len(spec) == 0 {
		// Fallback to default namespace
		if detectedNamespace != "default" {
			spec, err = h.parameterClient.GetSpec(ctx, instanceName, "default")
		}
		if err != nil || spec == nil {
			return nil, err
		}
	}
	return spec, nil
}

// getCRDSchemaRawWithFallback retrieves the raw CRD schema with fallback to sample schema.
// This is useful when you need the full CRD schema, not just the spec portion.
func (h *Handler) getCRDSchemaRawWithFallback(ctx context.Context, logger logr.Logger) map[string]interface{} {
	crdSchema, err := h.parameterClient.GetCRDSchema(ctx)
	if err != nil {
		logger.V(1).Info("failed to get CRD schema, using sample schema for local development", "error", err)
		crdSchema = GetSampleCRDSchema()
	}
	return crdSchema
}

