package api

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/garunski/conductor-framework/pkg/framework/reconciler"
	"gopkg.in/yaml.v3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// filterManifestsByServices filters manifests by service names
// It matches manifests to services based on resource name patterns
func filterManifestsByServices(manifests map[string][]byte, services []string) map[string][]byte {
	if len(services) == 0 {
		return manifests
	}

	// Create a set of service names for quick lookup
	serviceSet := make(map[string]bool)
	for _, svc := range services {
		serviceSet[svc] = true
	}

	filtered := make(map[string][]byte)

	for key, yamlData := range manifests {
		// Key format: namespace/kind/name
		parts := strings.Split(key, "/")
		if len(parts) < 3 {
			continue
		}

		name := parts[2]

		// Map resource names to service names
		serviceName := name
		if strings.HasSuffix(name, "-backend") {
			serviceName = strings.TrimSuffix(name, "-backend")
		} else if strings.HasSuffix(name, "-pvc") {
			serviceName = strings.TrimSuffix(name, "-pvc")
		} else if strings.HasSuffix(name, "-secrets") {
			serviceName = strings.TrimSuffix(name, "-secrets")
		} else if strings.HasSuffix(name, "-config") {
			serviceName = strings.TrimSuffix(name, "-config")
		}

		// Check if this resource belongs to any of the selected services
		matched := false
		for svc := range serviceSet {
			if strings.HasPrefix(serviceName, svc) || serviceName == svc {
				matched = true
				break
			}
		}

		if matched {
			filtered[key] = yamlData
		}
	}

	return filtered
}

// checkServiceInstalled checks if a service is installed by looking for its main workload (Deployment or StatefulSet)
func checkServiceInstalled(ctx context.Context, serviceName string, manifests map[string][]byte, rec reconciler.Reconciler) bool {
	if rec == nil {
		return false
	}

	clientset := rec.GetClientset()
	if clientset == nil {
		return false
	}

	// Filter manifests for this service
	serviceManifests := filterManifestsByServices(manifests, []string{serviceName})

	// Look for Deployment or StatefulSet resources
	for key := range serviceManifests {
		parts := strings.Split(key, "/")
		if len(parts) < 3 {
			continue
		}

		namespace := parts[0]
		if namespace == "" {
			namespace = "default"
		}
		kind := parts[1]
		name := parts[2]

		// Check if this is a Deployment or StatefulSet
		if kind == "Deployment" {
			_, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return true
			}
			if !k8serrors.IsNotFound(err) {
				// Log error but continue checking other resources
				continue
			}
		} else if kind == "StatefulSet" {
			_, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return true
			}
			if !k8serrors.IsNotFound(err) {
				// Log error but continue checking other resources
				continue
			}
		}
	}

	return false
}

// getServiceInstallationStatus returns a map of service name -> installation status
// Optimized to check services in parallel for better performance
func getServiceInstallationStatus(ctx context.Context, services []string, manifests map[string][]byte, rec reconciler.Reconciler) map[string]bool {
	statusMap := make(map[string]bool)
	if len(services) == 0 {
		return statusMap
	}

	// Create a context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Use a mutex to protect the statusMap
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Check all services in parallel
	for _, service := range services {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()
			installed := checkServiceInstalled(checkCtx, svc, manifests, rec)
			mu.Lock()
			statusMap[svc] = installed
			mu.Unlock()
		}(service)
	}

	wg.Wait()
	return statusMap
}

// updateManifestsWithCurrentParameters updates manifests with current parameters (especially namespace)
// This ensures that global defaults are applied when deploying
func (h *Handler) updateManifestsWithCurrentParameters(ctx context.Context, manifests map[string][]byte, instanceName string) (map[string][]byte, error) {
	defaultNamespace := "default"
	updatedManifests := make(map[string][]byte)
	
	// Group manifests by service name
	serviceManifests := make(map[string]map[string][]byte)
	
	for key, yamlData := range manifests {
		parts := strings.Split(key, "/")
		if len(parts) < 3 {
			updatedManifests[key] = yamlData
			continue
		}
		
		name := parts[2]
		serviceName := name
		if strings.HasSuffix(name, "-backend") {
			serviceName = strings.TrimSuffix(name, "-backend")
		} else if strings.HasSuffix(name, "-pvc") {
			serviceName = strings.TrimSuffix(name, "-pvc")
		} else if strings.HasSuffix(name, "-secrets") {
			serviceName = strings.TrimSuffix(name, "-secrets")
		} else if strings.HasSuffix(name, "-config") {
			serviceName = strings.TrimSuffix(name, "-config")
		}
		
		if serviceManifests[serviceName] == nil {
			serviceManifests[serviceName] = make(map[string][]byte)
		}
		serviceManifests[serviceName][key] = yamlData
	}
	
	// Get spec once for all services
	spec, err := h.parameterClient.GetSpec(ctx, instanceName, defaultNamespace)
	if err != nil {
		h.logger.V(1).Info("failed to get spec, using existing manifests", "error", err)
		// Use existing manifests if we can't get spec
		for _, serviceManifestsMap := range serviceManifests {
			for k, v := range serviceManifestsMap {
				updatedManifests[k] = v
			}
		}
		return updatedManifests, nil
	}
	
	// Update each service's manifests with current parameters
	for serviceName, serviceManifestsMap := range serviceManifests {
		// Determine target namespace from spec
		targetNamespace := "default"
		
		// Check global namespace first
		if spec != nil {
			if global, ok := spec["global"].(map[string]interface{}); ok {
				if ns, ok := global["namespace"].(string); ok && ns != "" {
					targetNamespace = ns
				}
			}
			
			// Override with service-specific namespace if present
			if services, ok := spec["services"].(map[string]interface{}); ok {
				if service, ok := services[serviceName].(map[string]interface{}); ok {
					if ns, ok := service["namespace"].(string); ok && ns != "" {
						targetNamespace = ns
					}
				}
			}
		}
		
		// Update each manifest in this service
		for key, yamlData := range serviceManifestsMap {
			// Parse YAML
			var obj map[string]interface{}
			if err := yaml.Unmarshal(yamlData, &obj); err != nil {
				h.logger.V(1).Info("failed to parse manifest, using as-is", "key", key, "error", err)
				updatedManifests[key] = yamlData
				continue
			}
			
			// Update namespace in metadata
			metadata, ok := obj["metadata"].(map[string]interface{})
			if !ok {
				updatedManifests[key] = yamlData
				continue
			}
			
			oldNamespace := "default"
			if ns, ok := metadata["namespace"].(string); ok && ns != "" {
				oldNamespace = ns
			}
			
			// Update namespace if it's different
			if oldNamespace != targetNamespace {
				metadata["namespace"] = targetNamespace
				
				// Re-marshal YAML
				updatedYAML, err := yaml.Marshal(obj)
				if err != nil {
					h.logger.V(1).Info("failed to marshal updated manifest, using as-is", "key", key, "error", err)
					updatedManifests[key] = yamlData
					continue
				}
				
				// Update key with new namespace
				parts := strings.Split(key, "/")
				if len(parts) >= 3 {
					newKey := fmt.Sprintf("%s/%s/%s", targetNamespace, parts[1], parts[2])
					updatedManifests[newKey] = updatedYAML
				} else {
					updatedManifests[key] = updatedYAML
				}
			} else {
				updatedManifests[key] = yamlData
			}
		}
	}
	
	return updatedManifests, nil
}

