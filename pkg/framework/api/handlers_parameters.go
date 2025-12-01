package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
	"github.com/garunski/conductor-framework/pkg/framework/manifest"
)

// GetParameters retrieves the current deployment parameters
func (h *Handler) GetParameters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Detect namespace from manifests
	manifests := h.store.List()
	detectedNamespace := detectNamespaceFromManifests(manifests)
	if detectedNamespace == "" {
		detectedNamespace = "default"
	}

	spec, err := h.parameterClient.GetSpec(ctx, crd.DefaultName, detectedNamespace)
	if err != nil || spec == nil || len(spec) == 0 {
		// Fallback to default namespace
		if detectedNamespace != "default" {
			spec, err = h.parameterClient.GetSpec(ctx, crd.DefaultName, "default")
		}
		if err != nil {
			h.logger.Error(err, "failed to get DeploymentParameters spec")
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "get_parameters_failed", err.Error(), nil)
			return
		}
	}
	if err != nil {
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "get_parameters_failed", err.Error(), nil)
		return
	}

	if spec == nil || len(spec) == 0 {
		// Return default parameters if CRD doesn't exist
		spec = map[string]interface{}{
			"global": map[string]interface{}{
				"namespace":  "default",
				"namePrefix": "",
				"replicas":   int32(1),
			},
		}
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, spec)
}

// UpdateParameters creates or updates deployment parameters
func (h *Handler) UpdateParameters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Detect namespace from manifests
	manifests := h.store.List()
	detectedNamespace := detectNamespaceFromManifests(manifests)
	if detectedNamespace == "" {
		detectedNamespace = "default"
	}

	var spec map[string]interface{}

	if err := h.parseJSONRequest(r, &spec); err != nil {
		WriteErrorResponse(w, h.logger, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	// Get existing parameters to check if it exists
	params, err := h.parameterClient.Get(ctx, crd.DefaultName, detectedNamespace)
	if err != nil {
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "get_parameters_failed", err.Error(), nil)
		return
	}

	if params == nil {
		// Create new
		if err := h.parameterClient.CreateWithSpec(ctx, crd.DefaultName, detectedNamespace, spec); err != nil {
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "create_parameters_failed", err.Error(), nil)
			return
		}
	} else {
		// Update existing
		if err := h.parameterClient.UpdateSpec(ctx, crd.DefaultName, detectedNamespace, spec); err != nil {
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "update_parameters_failed", err.Error(), nil)
			return
		}
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{"message": "Parameters updated successfully"})
}

// GetServiceParameters retrieves parameters for a specific service from the spec
func (h *Handler) GetServiceParameters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := chi.URLParam(r, "service")
	
	// Detect namespace from manifests
	manifests := h.store.List()
	detectedNamespace := detectNamespaceFromManifests(manifests)
	if detectedNamespace == "" {
		detectedNamespace = "default"
	}

	if serviceName == "" {
		WriteErrorResponse(w, h.logger, http.StatusBadRequest, "invalid_request", "service name is required", nil)
		return
	}

	spec, err := h.parameterClient.GetSpec(ctx, crd.DefaultName, detectedNamespace)
	if err != nil || spec == nil || len(spec) == 0 {
		// Fallback to default namespace
		if detectedNamespace != "default" {
			spec, err = h.parameterClient.GetSpec(ctx, crd.DefaultName, "default")
		}
		if err != nil {
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "get_service_parameters_failed", err.Error(), nil)
			return
		}
	}

	// Extract service from spec
	var serviceParams interface{}
	if spec != nil {
		services, ok := spec["services"].(map[string]interface{})
		if ok {
			serviceParams = services[serviceName]
		}
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, serviceParams)
}

func int32Ptr(i int32) *int32 {
	return &i
}

// GetServiceValues returns both merged/default values and actual deployed values for all services
func (h *Handler) GetServiceValues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Detect namespace from manifests
	manifests := h.store.List()
	detectedNamespace := detectNamespaceFromManifests(manifests)
	if detectedNamespace == "" {
		detectedNamespace = "default"
	}

	// Get all services using the same logic as the template
	services := h.getServiceNames()

	result := make(map[string]map[string]interface{})
	clientset := h.reconciler.GetClientset()
	allManifests := h.store.List()

	for _, serviceName := range services {
		serviceData := make(map[string]interface{})

		// Get merged/default values - always provide defaults even if cluster is unavailable
		var merged map[string]interface{}
		
		// Try to get spec, but don't fail if cluster is unavailable
		spec, err := h.parameterClient.GetSpec(ctx, crd.DefaultName, detectedNamespace)
		if err != nil || spec == nil || len(spec) == 0 {
			// Fallback to default namespace
			if detectedNamespace != "default" {
				spec, err = h.parameterClient.GetSpec(ctx, crd.DefaultName, "default")
			}
		}
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
			deployCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
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

// GetParametersSchema returns the CRD schema for form generation
func (h *Handler) GetParametersSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get CRD schema definition (raw OpenAPI schema) for form generation
	crdSchema, err := h.parameterClient.GetCRDSchema(ctx)
	if err != nil {
		h.logger.V(1).Info("failed to get CRD schema, using sample schema for local development", "error", err)
		// Use sample schema for local development/debugging
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
	
	if specSchema == nil {
		specSchema = make(map[string]interface{})
	}
	
	WriteJSONResponse(w, h.logger, http.StatusOK, specSchema)
}

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

func getDeployedValues(ctx context.Context, clientset kubernetes.Interface, serviceName, namespace string, manifests map[string][]byte) map[string]interface{} {
	if clientset == nil {
		return nil
	}

	// Try to find the deployment/statefulset for this service
	var deployment *appsv1.Deployment
	var statefulSet *appsv1.StatefulSet
	var err error

	apps := clientset.AppsV1()

	// Try to find the resource name from manifests
	resourceName := serviceName
	for key := range manifests {
		parts := strings.Split(key, "/")
		if len(parts) >= 3 {
			ns := parts[0]
			if ns == "" {
				ns = "default"
			}
			if ns == namespace {
				kind := parts[1]
				name := parts[2]
				if (kind == "Deployment" || kind == "StatefulSet") && strings.Contains(name, serviceName) {
					resourceName = name
					if kind == "Deployment" {
						deployment, err = apps.Deployments(namespace).Get(ctx, resourceName, metav1.GetOptions{})
						if err != nil {
							if k8serrors.IsNotFound(err) {
								deployment = nil
							} else {
								return nil
							}
						}
					} else {
						statefulSet, err = apps.StatefulSets(namespace).Get(ctx, resourceName, metav1.GetOptions{})
						if err != nil {
							if k8serrors.IsNotFound(err) {
								statefulSet = nil
							} else {
								return nil
							}
						}
					}
					break
				}
			}
		}
	}

	// If not found, try common naming patterns
	if deployment == nil && statefulSet == nil {
		// Try with service name directly
		deployment, err = apps.Deployments(namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				deployment = nil
			} else {
				return nil
			}
		}
		if deployment == nil {
			statefulSet, err = apps.StatefulSets(namespace).Get(ctx, serviceName, metav1.GetOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					statefulSet = nil
				} else {
					return nil
				}
			}
		}
	}

	// If no deployment or statefulset was found, return nil
	if deployment == nil && statefulSet == nil {
		return nil
	}

	result := make(map[string]interface{})

	if deployment != nil {
		result["namespace"] = deployment.Namespace
		if deployment.Spec.Replicas != nil {
			result["replicas"] = *deployment.Spec.Replicas
		}
		if len(deployment.Spec.Template.Spec.Containers) > 0 {
			container := deployment.Spec.Template.Spec.Containers[0]
			// Extract image tag
			if image := container.Image; image != "" {
				parts := strings.Split(image, ":")
				if len(parts) > 1 {
					result["imageTag"] = parts[len(parts)-1]
				}
			}
			// Extract resources
			if container.Resources.Requests != nil {
				requests := make(map[string]interface{})
				if mem := container.Resources.Requests[corev1.ResourceMemory]; !mem.IsZero() {
					requests["memory"] = mem.String()
				}
				if cpu := container.Resources.Requests[corev1.ResourceCPU]; !cpu.IsZero() {
					requests["cpu"] = cpu.String()
				}
				if len(requests) > 0 {
					result["resources"] = map[string]interface{}{"requests": requests}
				}
			}
			if container.Resources.Limits != nil {
				limits := make(map[string]interface{})
				if mem := container.Resources.Limits[corev1.ResourceMemory]; !mem.IsZero() {
					limits["memory"] = mem.String()
				}
				if cpu := container.Resources.Limits[corev1.ResourceCPU]; !cpu.IsZero() {
					limits["cpu"] = cpu.String()
				}
				if len(limits) > 0 {
					if result["resources"] == nil {
						result["resources"] = make(map[string]interface{})
					}
					result["resources"].(map[string]interface{})["limits"] = limits
				}
			}
		}
	} else if statefulSet != nil {
		result["namespace"] = statefulSet.Namespace
		if statefulSet.Spec.Replicas != nil {
			result["replicas"] = *statefulSet.Spec.Replicas
		}
		if len(statefulSet.Spec.Template.Spec.Containers) > 0 {
			container := statefulSet.Spec.Template.Spec.Containers[0]
			// Extract image tag
			if image := container.Image; image != "" {
				parts := strings.Split(image, ":")
				if len(parts) > 1 {
					result["imageTag"] = parts[len(parts)-1]
				}
			}
			// Extract resources
			if container.Resources.Requests != nil {
				requests := make(map[string]interface{})
				if mem := container.Resources.Requests[corev1.ResourceMemory]; !mem.IsZero() {
					requests["memory"] = mem.String()
				}
				if cpu := container.Resources.Requests[corev1.ResourceCPU]; !cpu.IsZero() {
					requests["cpu"] = cpu.String()
				}
				if len(requests) > 0 {
					result["resources"] = map[string]interface{}{"requests": requests}
				}
			}
			if container.Resources.Limits != nil {
				limits := make(map[string]interface{})
				if mem := container.Resources.Limits[corev1.ResourceMemory]; !mem.IsZero() {
					limits["memory"] = mem.String()
				}
				if cpu := container.Resources.Limits[corev1.ResourceCPU]; !cpu.IsZero() {
					limits["cpu"] = cpu.String()
				}
				if len(limits) > 0 {
					if result["resources"] == nil {
						result["resources"] = make(map[string]interface{})
					}
					result["resources"].(map[string]interface{})["limits"] = limits
				}
			}
		}
		// Extract storage size from volume claims
		if len(statefulSet.Spec.VolumeClaimTemplates) > 0 {
			for _, vct := range statefulSet.Spec.VolumeClaimTemplates {
				if storage := vct.Spec.Resources.Requests[corev1.ResourceStorage]; !storage.IsZero() {
					result["storageSize"] = storage.String()
					break
				}
			}
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

