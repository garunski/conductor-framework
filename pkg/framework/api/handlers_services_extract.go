package api

import (
	"context"
	"fmt"

	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"gopkg.in/yaml.v3"
)

// k8sObject represents a generic Kubernetes object
type k8sObject struct {
	Kind     string                 `yaml:"kind"`
	Metadata k8sMetadata            `yaml:"metadata"`
	Spec     map[string]interface{} `yaml:"spec"`
}

// k8sMetadata represents Kubernetes object metadata
type k8sMetadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// k8sServiceSpec represents a Service spec
type k8sServiceSpec struct {
	Ports    []k8sServicePort       `yaml:"ports"`
	Selector map[string]string      `yaml:"selector"`
	Other    map[string]interface{} `yaml:",inline"`
}

// k8sServicePort represents a Service port
type k8sServicePort struct {
	Port int `yaml:"port"`
}

// k8sDeploymentSpec represents a Deployment/StatefulSet spec
type k8sDeploymentSpec struct {
	Template k8sPodTemplateSpec     `yaml:"template"`
	Other    map[string]interface{} `yaml:",inline"`
}

// k8sPodTemplateSpec represents a pod template
type k8sPodTemplateSpec struct {
	Metadata k8sMetadata            `yaml:"metadata"`
	Spec     k8sPodSpec             `yaml:"spec"`
	Other    map[string]interface{} `yaml:",inline"`
}

// k8sPodSpec represents a pod spec
type k8sPodSpec struct {
	Containers []k8sContainer         `yaml:"containers"`
	Other      map[string]interface{} `yaml:",inline"`
}

// k8sContainer represents a container
type k8sContainer struct {
	Env   []k8sEnvVar            `yaml:"env"`
	Other map[string]interface{} `yaml:",inline"`
}

// k8sEnvVar represents an environment variable
type k8sEnvVar struct {
	Name      string                 `yaml:"name"`
	Value     string                 `yaml:"value"`
	ValueFrom map[string]interface{} `yaml:"valueFrom"`
}

func extractServices(ctx context.Context, manifests map[string][]byte) []serviceInfo {
	services := make([]serviceInfo, 0)

	for _, yamlData := range manifests {
		// Check for context cancellation during iteration
		select {
		case <-ctx.Done():
			return services
		default:
		}
		var obj k8sObject
		if err := yaml.Unmarshal(yamlData, &obj); err != nil {
			continue
		}

		if obj.Kind != "Service" {
			continue
		}

		if obj.Metadata.Name == "" {
			continue
		}

		namespace := "default"
		if obj.Metadata.Namespace != "" {
			namespace = obj.Metadata.Namespace
		}

		// Parse ports from spec
		if ports, ok := obj.Spec["ports"].([]interface{}); ok && len(ports) > 0 {
			if portObj, ok := ports[0].(map[string]interface{}); ok {
				port := 0
				if p, ok := portObj["port"].(int); ok {
					port = p
				} else if p, ok := portObj["port"].(float64); ok {
					port = int(p)
				}
				if port > 0 {
					services = append(services, serviceInfo{
						Name:      obj.Metadata.Name,
						Namespace: namespace,
						Port:      port,
					})
				}
			}
		}
	}

	return services
}

func extractServiceManifest(ctx context.Context, manifests map[string][]byte, namespace, name string) ([]byte, bool) {
	for _, yamlData := range manifests {
		// Check for context cancellation during iteration
		select {
		case <-ctx.Done():
			return nil, false
		default:
		}
		var obj k8sObject
		if err := yaml.Unmarshal(yamlData, &obj); err != nil {
			continue
		}

		if obj.Kind != "Service" {
			continue
		}

		if obj.Metadata.Name != name {
			continue
		}

		ns := "default"
		if obj.Metadata.Namespace != "" {
			ns = obj.Metadata.Namespace
		}

		if ns == namespace {
			return yamlData, true
		}
	}
	return nil, false
}

func extractServiceSelector(ctx context.Context, serviceManifest []byte) (map[string]string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	var serviceObj k8sObject
	if err := yaml.Unmarshal(serviceManifest, &serviceObj); err != nil {
		return nil, apperrors.WrapInvalidYAML(err, "failed to unmarshal service manifest")
	}

	selectorRaw, ok := serviceObj.Spec["selector"]
	if !ok {
		return nil, apperrors.WrapInvalid(nil, "service spec missing selector")
	}

	selector, ok := selectorRaw.(map[string]interface{})
	if !ok {
		return nil, apperrors.WrapInvalid(nil, "service selector is not a map")
	}

	result := make(map[string]string)
	for k, v := range selector {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}

	return result, nil
}

func findMatchingDeployment(ctx context.Context, manifests map[string][]byte, namespace string, selector map[string]string) []byte {
	for _, yamlData := range manifests {
		// Check for context cancellation during iteration
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		var obj k8sObject
		if err := yaml.Unmarshal(yamlData, &obj); err != nil {
			continue
		}

		if obj.Kind != "Deployment" && obj.Kind != "StatefulSet" {
			continue
		}

		ns := "default"
		if obj.Metadata.Namespace != "" {
			ns = obj.Metadata.Namespace
		}

		if ns != namespace {
			continue
		}

		// Check if labels match selector
		if template, ok := obj.Spec["template"].(map[string]interface{}); ok {
			if metadata, ok := template["metadata"].(map[string]interface{}); ok {
				if labels, ok := metadata["labels"].(map[string]interface{}); ok {
					matches := true
					for key, value := range selector {
						if labelValue, ok := labels[key].(string); !ok || labelValue != value {
							matches = false
							break
						}
					}
					if matches {
						return yamlData
					}
				}
			}
		}
	}
	return nil
}

func extractEnvVars(ctx context.Context, deploymentManifest []byte) []EnvVar {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil
	default:
	}
	var deploymentObj k8sObject
	if err := yaml.Unmarshal(deploymentManifest, &deploymentObj); err != nil {
		return nil
	}

	var envVars []EnvVar

	if template, ok := deploymentObj.Spec["template"].(map[string]interface{}); ok {
		if spec, ok := template["spec"].(map[string]interface{}); ok {
			if containers, ok := spec["containers"].([]interface{}); ok {
				for _, containerRaw := range containers {
					if container, ok := containerRaw.(map[string]interface{}); ok {
						if env, ok := container["env"].([]interface{}); ok {
							for _, envRaw := range env {
								if envMap, ok := envRaw.(map[string]interface{}); ok {
									name, _ := envMap["name"].(string)
									if name == "" {
										continue
									}

									envVar := EnvVar{Name: name}

									// Handle direct value
									if value, ok := envMap["value"].(string); ok {
										envVar.Value = value
									}

									// Handle valueFrom (secret/configmap)
									if valueFrom, ok := envMap["valueFrom"].(map[string]interface{}); ok {
										if secretKeyRef, ok := valueFrom["secretKeyRef"].(map[string]interface{}); ok {
											secretName, _ := secretKeyRef["name"].(string)
											key, _ := secretKeyRef["key"].(string)
											if secretName != "" && key != "" {
												envVar.Source = fmt.Sprintf("secret:%s/%s", secretName, key)
											}
										} else if configMapKeyRef, ok := valueFrom["configMapKeyRef"].(map[string]interface{}); ok {
											configMapName, _ := configMapKeyRef["name"].(string)
											key, _ := configMapKeyRef["key"].(string)
											if configMapName != "" && key != "" {
												envVar.Source = fmt.Sprintf("configmap:%s/%s", configMapName, key)
											}
										}
									}

									envVars = append(envVars, envVar)
								}
							}
						}
					}
				}
			}
		}
	}

	return envVars
}

