package api

import (
	"fmt"
	"strings"
)

// findServiceManifests searches for manifests matching the service name
// Returns the manifest YAML bytes, preferring StatefulSet over Deployment if both exist
func (h *Handler) findServiceManifests(serviceName string) ([]byte, error) {
	manifests := h.store.List()
	
	var statefulSetManifest []byte
	var deploymentManifest []byte
	
	for key, manifestData := range manifests {
		parts := strings.Split(key, "/")
		if len(parts) >= 3 {
			kind := parts[1]
			name := parts[2]
			
			// Check if this manifest belongs to the service
			// Service names can be part of the resource name (e.g., "redis", "postgresql-clickhouse")
			if strings.Contains(strings.ToLower(name), strings.ToLower(serviceName)) ||
				strings.Contains(strings.ToLower(serviceName), strings.ToLower(name)) {
				
				if kind == "StatefulSet" {
					statefulSetManifest = manifestData
				} else if kind == "Deployment" {
					deploymentManifest = manifestData
				}
			}
		}
	}
	
	// Prefer StatefulSet over Deployment
	if statefulSetManifest != nil {
		return statefulSetManifest, nil
	}
	if deploymentManifest != nil {
		return deploymentManifest, nil
	}
	
	return nil, fmt.Errorf("no manifest found for service: %s", serviceName)
}

