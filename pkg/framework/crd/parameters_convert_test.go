package crd

import (
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestDeploymentParametersToUnstructured(t *testing.T) {
	logger := logr.Discard()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	client := NewClient(dynamicClient, logger, "conductor.io", "v1alpha1", "deploymentparameters")

	params := &DeploymentParameters{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-instance",
			Namespace:       "default",
			ResourceVersion: "123",
		},
		Spec: map[string]interface{}{
			"global": map[string]interface{}{
				"namespace": "default",
			},
		},
	}

	obj := client.deploymentParametersToUnstructured(params)

	if obj.GetName() != "test-instance" {
		t.Errorf("deploymentParametersToUnstructured() name = %v, want test-instance", obj.GetName())
	}

	if obj.GetNamespace() != "default" {
		t.Errorf("deploymentParametersToUnstructured() namespace = %v, want default", obj.GetNamespace())
	}

	if obj.GetResourceVersion() != "123" {
		t.Errorf("deploymentParametersToUnstructured() resourceVersion = %v, want 123", obj.GetResourceVersion())
	}

	gvk := obj.GroupVersionKind()
	if gvk.Kind != "DeploymentParameters" {
		t.Errorf("deploymentParametersToUnstructured() kind = %v, want DeploymentParameters", gvk.Kind)
	}

	// Verify spec was set (it's stored directly in Object)
	specRaw, exists := obj.Object["spec"]
	if !exists {
		t.Error("deploymentParametersToUnstructured() spec is missing")
		return
	}

	// DeploymentParametersSpec is a type alias for map[string]interface{}
	// We need to convert it properly
	var spec map[string]interface{}
	switch v := specRaw.(type) {
	case map[string]interface{}:
		spec = v
	case DeploymentParametersSpec:
		spec = map[string]interface{}(v)
	default:
		t.Errorf("deploymentParametersToUnstructured() spec is not a map, got type %T", specRaw)
		return
	}

	// Verify the spec content
	global, ok := spec["global"].(map[string]interface{})
	if !ok {
		t.Errorf("deploymentParametersToUnstructured() spec.global is not a map, got type %T", spec["global"])
		return
	}

	if global["namespace"] != "default" {
		t.Errorf("deploymentParametersToUnstructured() spec.global.namespace = %v, want default", global["namespace"])
	}
}

func TestUnstructuredToDeploymentParameters(t *testing.T) {
	logger := logr.Discard()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	client := NewClient(dynamicClient, logger, "conductor.io", "v1alpha1", "deploymentparameters")

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "conductor.io/v1alpha1",
			"kind":       "DeploymentParameters",
			"metadata": map[string]interface{}{
				"name":            "test-instance",
				"namespace":       "default",
				"resourceVersion": "123",
			},
			"spec": map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": "default",
				},
			},
		},
	}

	params, err := client.unstructuredToDeploymentParameters(obj)
	if err != nil {
		t.Fatalf("unstructuredToDeploymentParameters() error = %v", err)
	}

	if params.Name != "test-instance" {
		t.Errorf("unstructuredToDeploymentParameters() name = %v, want test-instance", params.Name)
	}

	if params.Namespace != "default" {
		t.Errorf("unstructuredToDeploymentParameters() namespace = %v, want default", params.Namespace)
	}

	if params.ResourceVersion != "123" {
		t.Errorf("unstructuredToDeploymentParameters() resourceVersion = %v, want 123", params.ResourceVersion)
	}

	if global, ok := params.Spec["global"].(map[string]interface{}); !ok {
		t.Error("unstructuredToDeploymentParameters() spec.global is not a map")
	} else {
		if global["namespace"] != "default" {
			t.Errorf("unstructuredToDeploymentParameters() spec.global.namespace = %v, want default", global["namespace"])
		}
	}
}

func TestUnstructuredToDeploymentParameters_NoSpec(t *testing.T) {
	logger := logr.Discard()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	client := NewClient(dynamicClient, logger, "conductor.io", "v1alpha1", "deploymentparameters")

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "conductor.io/v1alpha1",
			"kind":       "DeploymentParameters",
			"metadata": map[string]interface{}{
				"name":      "test-instance",
				"namespace": "default",
			},
		},
	}

	params, err := client.unstructuredToDeploymentParameters(obj)
	if err != nil {
		t.Fatalf("unstructuredToDeploymentParameters() error = %v", err)
	}

	if params.Spec == nil {
		t.Error("unstructuredToDeploymentParameters() spec should not be nil")
	}

	if len(params.Spec) != 0 {
		t.Errorf("unstructuredToDeploymentParameters() spec should be empty, got %v", params.Spec)
	}
}

func TestDeepCopyMap(t *testing.T) {
	src := map[string]interface{}{
		"nested": map[string]interface{}{
			"value": "test",
		},
		"slice": []interface{}{1, 2, 3},
		"string": "test",
		"number": 42,
	}

	dst := deepCopyMap(src)

	// Modify original
	src["nested"].(map[string]interface{})["value"] = "modified"
	src["slice"].([]interface{})[0] = 999
	src["string"] = "modified"
	src["number"] = 999

	// Check that copy wasn't affected
	if dst["nested"].(map[string]interface{})["value"] != "test" {
		t.Error("deepCopyMap() did not create a deep copy of nested map")
	}
	if dst["slice"].([]interface{})[0] != 1 {
		t.Error("deepCopyMap() did not create a deep copy of slice")
	}
	if dst["string"] != "test" {
		t.Error("deepCopyMap() did not copy string value")
	}
	if dst["number"] != 42 {
		t.Error("deepCopyMap() did not copy number value")
	}
}

func TestDeepCopyValue(t *testing.T) {
	// Test map
	srcMap := map[string]interface{}{"key": "value"}
	dstMap := deepCopyValue(srcMap).(map[string]interface{})
	dstMap["newkey"] = "newvalue"
	if _, exists := srcMap["newkey"]; exists {
		t.Error("deepCopyValue() did not create a deep copy of map")
	}

	// Test slice
	srcSlice := []interface{}{1, 2, 3}
	dstSlice := deepCopyValue(srcSlice).([]interface{})
	dstSlice[0] = 999
	if srcSlice[0] == 999 {
		t.Error("deepCopyValue() did not create a deep copy of slice")
	}

	// Test nested map
	srcNested := map[string]interface{}{
		"nested": map[string]interface{}{
			"deep": "value",
		},
	}
	dstNested := deepCopyValue(srcNested).(map[string]interface{})
	dstNested["nested"].(map[string]interface{})["deep"] = "modified"
	if srcNested["nested"].(map[string]interface{})["deep"] == "modified" {
		t.Error("deepCopyValue() did not create a deep copy of nested map")
	}

	// Test primitives (should return as-is)
	if deepCopyValue("test") != "test" {
		t.Error("deepCopyValue() should return string as-is")
	}
	if deepCopyValue(42) != 42 {
		t.Error("deepCopyValue() should return number as-is")
	}
	if deepCopyValue(true) != true {
		t.Error("deepCopyValue() should return bool as-is")
	}
}

