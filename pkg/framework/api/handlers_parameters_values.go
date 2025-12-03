package api

import (
	"context"
	"net/http"

	"github.com/garunski/conductor-framework/pkg/framework/manifest"
)

// GetServiceValues returns both merged/default values and actual deployed values for all services
func (h *Handler) GetServiceValues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get namespace and instance name
	detectedNamespace, instanceName := h.getNamespaceAndInstance(r)

	// Get all services using the same logic as the template
	services := h.getServiceNames()

	result := make(map[string]map[string]interface{})
	clientset := h.reconciler.GetClientset()
	allManifests := h.store.List()

	for _, serviceName := range services {
		serviceData := make(map[string]interface{})

		// Get merged/default values - always provide defaults even if cluster is unavailable
		var merged map[string]interface{}
		
		// Try to get spec with fallback, but don't fail if cluster is unavailable
		spec, err := h.getSpecWithFallback(ctx, instanceName, detectedNamespace)
		if err == nil && spec != nil {
			// Merge global and service-specific parameters
			merged = make(map[string]interface{})
			
			// Start with global defaults
			if global, ok := spec["global"].(map[string]interface{}); ok {
				for k, v := range global {
					merged[k] = v
				}
			}
			
			// Apply service-specific overrides
			if services, ok := spec["services"].(map[string]interface{}); ok {
				if service, ok := services[serviceName].(map[string]interface{}); ok {
					for k, v := range service {
						merged[k] = v
					}
				}
			}
		}
		
		// If no merged params (cluster unavailable or no saved params), try to extract from manifests
		if merged == nil || len(merged) == 0 {
			// Try to get defaults from manifests
			manifestYAML, err := h.findServiceManifests(serviceName)
			if err == nil && manifestYAML != nil {
				manifestDefaults, err := manifest.ExtractDefaultsFromManifest(manifestYAML, serviceName)
				if err == nil && manifestDefaults != nil {
					// manifestDefaults is already a map[string]interface{}
					merged = manifestDefaults
				}
			}
			
			// Fall back to hardcoded defaults if manifest extraction failed
			if merged == nil || len(merged) == 0 {
				merged = map[string]interface{}{
					"namespace":  "default",
					"namePrefix": "",
					"replicas":   1,
					"storageSize": "",
					"imageTag":   "",
				}
			}
		}
		
		serviceData["merged"] = merged

		// Get actual deployed values from Kubernetes (only if cluster is available)
		// Silently ignore errors if cluster is not available
		if clientset != nil {
			// Use a short timeout context to avoid hanging if cluster is unavailable
			deployCtx, cancel := context.WithTimeout(ctx, DefaultHealthCheckTimeout)
			deployed := getDeployedValues(deployCtx, clientset, serviceName, detectedNamespace, allManifests)
			cancel()
			if deployed != nil {
				serviceData["deployed"] = deployed
			}
		}

		// Always add service data with at least default/merged values
		result[serviceName] = serviceData
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, result)
}

