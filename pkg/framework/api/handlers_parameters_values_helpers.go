package api

import (
	"context"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func getDeployedValues(ctx context.Context, clientset kubernetes.Interface, serviceName, namespace string, manifests map[string][]byte) map[string]interface{} {
	if clientset == nil {
		return nil
	}

	deployment, statefulSet := findDeploymentOrStatefulSet(ctx, clientset, serviceName, namespace, manifests)
	if deployment == nil && statefulSet == nil {
		return nil
	}

	if deployment != nil {
		return extractDeploymentValues(deployment)
	}
	return extractStatefulSetValues(statefulSet)
}

func findDeploymentOrStatefulSet(ctx context.Context, clientset kubernetes.Interface, serviceName, namespace string, manifests map[string][]byte) (*appsv1.Deployment, *appsv1.StatefulSet) {
	apps := clientset.AppsV1()

	// Try to find the resource name from manifests
	resourceName := serviceName
	var deployment *appsv1.Deployment
	var statefulSet *appsv1.StatefulSet
	var err error

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
								return nil, nil
							}
						}
					} else {
						statefulSet, err = apps.StatefulSets(namespace).Get(ctx, resourceName, metav1.GetOptions{})
						if err != nil {
							if k8serrors.IsNotFound(err) {
								statefulSet = nil
							} else {
								return nil, nil
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
		deployment, err = apps.Deployments(namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				deployment = nil
			} else {
				return nil, nil
			}
		}
		if deployment == nil {
			statefulSet, err = apps.StatefulSets(namespace).Get(ctx, serviceName, metav1.GetOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					statefulSet = nil
				} else {
					return nil, nil
				}
			}
		}
	}

	return deployment, statefulSet
}

func extractDeploymentValues(deployment *appsv1.Deployment) map[string]interface{} {
	result := make(map[string]interface{})
	result["namespace"] = deployment.Namespace

	if deployment.Spec.Replicas != nil {
		result["replicas"] = *deployment.Spec.Replicas
	}

	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		container := deployment.Spec.Template.Spec.Containers[0]
		extractContainerValues(container, result)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func extractStatefulSetValues(statefulSet *appsv1.StatefulSet) map[string]interface{} {
	result := make(map[string]interface{})
	result["namespace"] = statefulSet.Namespace

	if statefulSet.Spec.Replicas != nil {
		result["replicas"] = *statefulSet.Spec.Replicas
	}

	if len(statefulSet.Spec.Template.Spec.Containers) > 0 {
		container := statefulSet.Spec.Template.Spec.Containers[0]
		extractContainerValues(container, result)
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

	if len(result) == 0 {
		return nil
	}
	return result
}

func extractContainerValues(container corev1.Container, result map[string]interface{}) {
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

