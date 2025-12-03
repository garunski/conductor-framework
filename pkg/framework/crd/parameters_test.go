package crd

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestGetCRDSchema(t *testing.T) {
	logger := logr.Discard()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	client := NewClient(dynamicClient, logger, "conductor.io", "v1alpha1", "deploymentparameters")

	// Create a mock CRD object
	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	crdInterface := dynamicClient.Resource(crdGVR)

	crdName := "deploymentparameters.conductor.io"
	crdObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io/v1",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name": crdName,
			},
			"spec": map[string]interface{}{
				"versions": []interface{}{
					map[string]interface{}{
						"name": "v1alpha1",
						"schema": map[string]interface{}{
							"openAPIV3Schema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"spec": map[string]interface{}{
										"type": "object",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	_, err := crdInterface.Create(ctx, crdObj, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create mock CRD: %v", err)
	}

	// Test GetCRDSchema
	schema, err := client.GetCRDSchema(ctx)
	if err != nil {
		t.Fatalf("GetCRDSchema() error = %v", err)
	}

	if schema == nil {
		t.Error("GetCRDSchema() returned nil schema")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Error("GetCRDSchema() schema does not have properties")
	}

	if _, ok := properties["spec"]; !ok {
		t.Error("GetCRDSchema() schema does not have spec property")
	}
}

func TestGetCRDSchema_NotFound(t *testing.T) {
	logger := logr.Discard()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	client := NewClient(dynamicClient, logger, "conductor.io", "v1alpha1", "deploymentparameters")

	ctx := context.Background()
	_, err := client.GetCRDSchema(ctx)
	if err == nil {
		t.Error("GetCRDSchema() expected error for non-existent CRD, got nil")
	}
}

func TestGetSpecSchema(t *testing.T) {
	logger := logr.Discard()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	client := NewClient(dynamicClient, logger, "conductor.io", "v1alpha1", "deploymentparameters")

	// Create a mock CRD object with full schema
	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	crdInterface := dynamicClient.Resource(crdGVR)

	crdName := "deploymentparameters.conductor.io"
	crdObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io/v1",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name": crdName,
			},
			"spec": map[string]interface{}{
				"versions": []interface{}{
					map[string]interface{}{
						"name": "v1alpha1",
						"schema": map[string]interface{}{
							"openAPIV3Schema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"spec": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"global": map[string]interface{}{
												"type": "object",
												"properties": map[string]interface{}{
													"namespace": map[string]interface{}{
														"type": "string",
													},
												},
											},
											"services": map[string]interface{}{
												"type": "object",
												"additionalProperties": map[string]interface{}{
													"type": "object",
													"additionalProperties": true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	_, err := crdInterface.Create(ctx, crdObj, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create mock CRD: %v", err)
	}

	// Test GetSpecSchema
	specSchema, err := client.GetSpecSchema(ctx)
	if err != nil {
		t.Fatalf("GetSpecSchema() error = %v", err)
	}

	if specSchema == nil {
		t.Error("GetSpecSchema() returned nil")
	}

	global, ok := specSchema["global"].(map[string]interface{})
	if !ok {
		t.Error("GetSpecSchema() does not have global")
	}

	if _, ok := global["namespace"]; !ok {
		t.Error("GetSpecSchema() global does not have namespace")
	}

	services, ok := specSchema["services"].(map[string]interface{})
	if !ok {
		t.Error("GetSpecSchema() does not have services")
	}

	// Services with additionalProperties: true should return empty map
	if len(services) != 0 {
		t.Errorf("GetSpecSchema() services should be empty map for additionalProperties: true, got %v", services)
	}
}

func TestExtractSchemaStructure(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "simple properties",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"field1": map[string]interface{}{
						"type": "string",
					},
					"field2": map[string]interface{}{
						"type": "number",
					},
				},
			},
			expected: map[string]interface{}{
				"field1": "string",
				"field2": "number",
			},
		},
		{
			name: "additionalProperties true",
			schema: map[string]interface{}{
				"type":                 "object",
				"additionalProperties": true,
			},
			expected: map[string]interface{}{},
		},
		{
			name: "additionalProperties with type",
			schema: map[string]interface{}{
				"type": "object",
				"additionalProperties": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"nested": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"nested": "string",
			},
		},
		{
			name: "nested properties",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"config": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"database": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"host": map[string]interface{}{
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"config": map[string]interface{}{
					"database": map[string]interface{}{
						"host": "string",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSchemaStructure(tt.schema)
			if !mapsEqual(result, tt.expected) {
				t.Errorf("extractSchemaStructure() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// mapsEqual compares two maps recursively
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok {
			return false
		} else {
			if aMap, ok := v.(map[string]interface{}); ok {
				if bMap, ok := bv.(map[string]interface{}); ok {
					if !mapsEqual(aMap, bMap) {
						return false
					}
				} else {
					return false
				}
			} else if v != bv {
				return false
			}
		}
	}
	return true
}

// Note: Tests for Get, GetSpec, Create, CreateWithSpec, Update, UpdateSpec, List, CreateOrUpdate
// require proper CRD resource type registration in the fake dynamic client.
// The fake client panics when trying to list/create/update CRD resources without proper scheme setup.
// These operations are tested indirectly through integration tests and API handler tests.

// Note: deepCopyMap and deepCopyValue are unexported functions.
// They are tested indirectly through unstructuredToDeploymentParameters which uses deepCopyMap.
// Direct unit tests for these functions would require making them exported or using reflection.

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

// Note: Tests for Get, GetSpec, Create, CreateWithSpec, Update, UpdateSpec, List, CreateOrUpdate
// require proper CRD resource type registration in the fake dynamic client using NewSimpleDynamicClientWithCustomListKinds.
// These are complex to set up and are better tested through integration tests.
// The conversion functions above are tested directly.

