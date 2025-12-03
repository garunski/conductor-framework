package api

import (
	"context"
	"sort"
	"strings"
	"time"
)

// getServiceNames extracts service names from manifest keys
// Service names are derived from the manifest directory structure
func (h *Handler) getServiceNames() []string {
	manifests := h.store.List()
	serviceSet := make(map[string]bool)
	
	for key := range manifests {
		// Keys are in format "namespace/Kind/name"
		// For services, we look for Service kind resources
		parts := strings.Split(key, "/")
		if len(parts) >= 3 {
			kind := parts[1]
			name := parts[2]
			
			// Extract service name from resource name
			// Service names are typically the base name before any suffixes
			if kind == "Service" {
				// Remove common suffixes like "-service", "-svc"
				serviceName := strings.TrimSuffix(name, "-service")
				serviceName = strings.TrimSuffix(serviceName, "-svc")
				if serviceName != "" {
					serviceSet[serviceName] = true
				}
			} else if kind == "Deployment" || kind == "StatefulSet" {
				// Also consider Deployments and StatefulSets as services
				// Remove common suffixes
				serviceName := strings.TrimSuffix(name, "-deployment")
				serviceName = strings.TrimSuffix(serviceName, "-statefulset")
				serviceName = strings.TrimSuffix(serviceName, "-deploy")
				if serviceName != "" {
					serviceSet[serviceName] = true
				}
			}
		}
	}
	
	// Convert set to sorted slice
	services := make([]string, 0, len(serviceSet))
	for service := range serviceSet {
		services = append(services, service)
	}
	sort.Strings(services)
	
	return services
}

// getServiceValuesMap returns merged/default and deployed values for all services
// Similar to GetServiceValues but returns a map instead of writing HTTP response
func (h *Handler) getServiceValuesMap(ctx context.Context, services []string, defaultNamespace string, instanceName string) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})
	clientset := h.reconciler.GetClientset()
	manifests := h.store.List()

	for _, serviceName := range services {
		serviceData := make(map[string]interface{})

		// Get merged/default values
		var merged map[string]interface{}

		// Try to get spec, but don't fail if cluster is unavailable
		spec, err := h.parameterClient.GetSpec(ctx, instanceName, defaultNamespace)
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

		// Fall back to empty map if no merged params
		if merged == nil || len(merged) == 0 {
			merged = make(map[string]interface{})
		}

		serviceData["merged"] = merged

		// Get actual deployed values from Kubernetes (only if cluster is available)
		if clientset != nil {
			deployCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			deployed := getDeployedValues(deployCtx, clientset, serviceName, defaultNamespace, manifests)
			cancel()
			if deployed != nil {
				serviceData["deployed"] = deployed
			}
		}

		result[serviceName] = serviceData
	}

	return result
}

// detectNamespaceFromManifests extracts the most common namespace from manifest keys
// Manifest keys are in format "namespace/Kind/name"
func detectNamespaceFromManifests(manifests map[string][]byte) string {
	if len(manifests) == 0 {
		return ""
	}
	
	namespaceCounts := make(map[string]int)
	for key := range manifests {
		parts := strings.Split(key, "/")
		if len(parts) >= 1 && parts[0] != "" {
			namespaceCounts[parts[0]]++
		}
	}
	
	// Find the most common namespace
	maxCount := 0
	mostCommon := ""
	for ns, count := range namespaceCounts {
		if count > maxCount {
			maxCount = count
			mostCommon = ns
		}
	}
	
	return mostCommon
}

