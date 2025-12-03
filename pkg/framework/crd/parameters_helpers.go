package crd

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (c *Client) deploymentParametersToUnstructured(params *DeploymentParameters) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   c.group,
		Version: c.version,
		Kind:    "DeploymentParameters",
	})
	obj.SetName(params.Name)
	obj.SetNamespace(params.Namespace)
	if params.ResourceVersion != "" {
		obj.SetResourceVersion(params.ResourceVersion)
	}

	// Spec is already a map[string]interface{}, use it directly
	if params.Spec != nil {
		obj.Object["spec"] = params.Spec
	} else {
		obj.Object["spec"] = make(map[string]interface{})
	}

	return obj
}

func (c *Client) unstructuredToDeploymentParameters(obj *unstructured.Unstructured) (*DeploymentParameters, error) {
	params := &DeploymentParameters{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s", c.group, c.version),
			Kind:       "DeploymentParameters",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            obj.GetName(),
			Namespace:       obj.GetNamespace(),
			ResourceVersion: obj.GetResourceVersion(),
		},
	}

	// Extract spec directly as map[string]interface{} to preserve all fields
	specRaw, ok := obj.Object["spec"]
	if !ok {
		params.Spec = make(map[string]interface{})
		return params, nil
	}

	specMap, ok := specRaw.(map[string]interface{})
	if !ok {
		params.Spec = make(map[string]interface{})
		return params, nil
	}

	// Deep copy the spec map to avoid sharing references
	params.Spec = deepCopyMap(specMap)

	return params, nil
}

// deepCopyMap creates a deep copy of a map[string]interface{}
func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		dst[k] = deepCopyValue(v)
	}
	return dst
}

// deepCopyValue creates a deep copy of a value
func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return deepCopyMap(val)
	case []interface{}:
		dst := make([]interface{}, len(val))
		for i, item := range val {
			dst[i] = deepCopyValue(item)
		}
		return dst
	default:
		// For primitives (string, int, bool, etc.), return as-is
		return v
	}
}

