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

