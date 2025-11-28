package manifest

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"gopkg.in/yaml.v3"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

// ExtractDefaultsFromManifest extracts default parameter values from a Kubernetes manifest YAML
// It parses the YAML and extracts namespace, replicas, image tag, storage size, and resource requests/limits
func ExtractDefaultsFromManifest(manifestYAML []byte, serviceName string) (*crd.ParameterSet, error) {
	// Try to parse as Deployment first
	var deployment appsv1.Deployment
	if err := yaml.Unmarshal(manifestYAML, &deployment); err == nil && deployment.Kind == "Deployment" {
		return extractFromDeployment(&deployment), nil
	}

	// Try to parse as StatefulSet
	var statefulSet appsv1.StatefulSet
	if err := yaml.Unmarshal(manifestYAML, &statefulSet); err == nil && statefulSet.Kind == "StatefulSet" {
		return extractFromStatefulSet(&statefulSet), nil
	}

	return nil, fmt.Errorf("manifest is neither a Deployment nor StatefulSet")
}

func extractFromDeployment(dep *appsv1.Deployment) *crd.ParameterSet {
	params := &crd.ParameterSet{}

	// Extract namespace
	if dep.Namespace != "" {
		params.Namespace = dep.Namespace
	} else {
		params.Namespace = "default"
	}

	// Extract replicas
	if dep.Spec.Replicas != nil {
		params.Replicas = dep.Spec.Replicas
	} else {
		replicas := int32(1)
		params.Replicas = &replicas
	}

	// Extract image tag and resources from first container
	if len(dep.Spec.Template.Spec.Containers) > 0 {
		container := dep.Spec.Template.Spec.Containers[0]
		
		// Extract image tag
		if image := container.Image; image != "" {
			parts := splitImage(image)
			if len(parts) == 2 {
				params.ImageTag = parts[1]
			}
		}

		// Extract resources
		if container.Resources.Requests != nil || container.Resources.Limits != nil {
			params.Resources = &crd.ResourceRequirements{}
			
			if container.Resources.Requests != nil {
				params.Resources.Requests = &crd.ResourceList{
					Memory: resourceToString(container.Resources.Requests[corev1.ResourceMemory]),
					CPU:    resourceToString(container.Resources.Requests[corev1.ResourceCPU]),
				}
			}
			
			if container.Resources.Limits != nil {
				params.Resources.Limits = &crd.ResourceList{
					Memory: resourceToString(container.Resources.Limits[corev1.ResourceMemory]),
					CPU:    resourceToString(container.Resources.Limits[corev1.ResourceCPU]),
				}
			}
		}
	}

	return params
}

func extractFromStatefulSet(ss *appsv1.StatefulSet) *crd.ParameterSet {
	params := extractFromDeployment(&appsv1.Deployment{
		ObjectMeta: ss.ObjectMeta,
		Spec: appsv1.DeploymentSpec{
			Replicas: ss.Spec.Replicas,
			Template: ss.Spec.Template,
		},
	})

	// Extract storage size from volume claim templates
	if len(ss.Spec.VolumeClaimTemplates) > 0 {
		for _, vct := range ss.Spec.VolumeClaimTemplates {
			if vct.Spec.Resources.Requests != nil {
				if storage := vct.Spec.Resources.Requests[corev1.ResourceStorage]; !storage.IsZero() {
					params.StorageSize = storage.String()
					break
				}
			}
		}
	}

	return params
}

// splitImage splits an image string into repository and tag
// e.g., "redis:7-alpine" -> ["redis", "7-alpine"]
// e.g., "clickhouse/clickhouse-server:24.1" -> ["clickhouse/clickhouse-server", "24.1"]
func splitImage(image string) []string {
	// Split on last ":" to handle cases like "registry:5000/repo:tag"
	lastColon := strings.LastIndex(image, ":")
	if lastColon == -1 {
		return []string{image, ""}
	}
	
	return []string{image[:lastColon], image[lastColon+1:]}
}

// resourceToString converts a resource.Quantity to string, returns empty string if zero
func resourceToString(q resource.Quantity) string {
	if q.IsZero() {
		return ""
	}
	return q.String()
}

