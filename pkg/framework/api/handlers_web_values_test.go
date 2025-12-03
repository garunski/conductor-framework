package api

import (
	"context"
	"testing"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

func TestGetServiceValuesMap(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Create a CRD spec with global and service-specific parameters
	// Use float64 for numbers to be compatible with JSON unmarshaling
	ctx := context.Background()
	spec := map[string]interface{}{
		"global": map[string]interface{}{
			"namespace": "default",
			"replicas":  float64(1),
			"imageTag":  "latest",
		},
		"services": map[string]interface{}{
			"service1": map[string]interface{}{
				"replicas": float64(3),
				"imageTag": "v1.0.0",
			},
			"service2": map[string]interface{}{
				"config": map[string]interface{}{
					"enabled": true,
				},
			},
		},
	}

	err = handler.parameterClient.CreateWithSpec(ctx, crd.DefaultName, "default", spec)
	if err != nil {
		t.Fatalf("failed to create CRD spec: %v", err)
	}

	services := []string{"service1", "service2"}
	result := handler.getServiceValuesMap(ctx, services, "default", "default")

	// Check service1 - should have merged values (global + service-specific)
	if service1Data, ok := result["service1"]; !ok {
		t.Error("getServiceValuesMap() should return data for service1")
	} else {
		merged, ok := service1Data["merged"].(map[string]interface{})
		if !ok {
			t.Error("getServiceValuesMap() should return merged values as map")
		} else {
			// Should have global defaults
			if merged["namespace"] != "default" {
				t.Errorf("getServiceValuesMap() merged namespace = %v, want %v", merged["namespace"], "default")
			}
			// Should have service-specific override for replicas
			// Note: JSON unmarshaling converts numbers to float64
			if replicas, ok := merged["replicas"].(float64); !ok || replicas != 3 {
				t.Errorf("getServiceValuesMap() merged replicas = %v, want %v", merged["replicas"], float64(3))
			}
			// Should have service-specific override for imageTag
			if merged["imageTag"] != "v1.0.0" {
				t.Errorf("getServiceValuesMap() merged imageTag = %v, want %v", merged["imageTag"], "v1.0.0")
			}
		}
	}

	// Check service2 - should have merged values (global + service-specific)
	if service2Data, ok := result["service2"]; !ok {
		t.Error("getServiceValuesMap() should return data for service2")
	} else {
		merged, ok := service2Data["merged"].(map[string]interface{})
		if !ok {
			t.Error("getServiceValuesMap() should return merged values as map")
		} else {
			// Should have global defaults
			if merged["namespace"] != "default" {
				t.Errorf("getServiceValuesMap() merged namespace = %v, want %v", merged["namespace"], "default")
			}
			// Should have service-specific config
			if config, ok := merged["config"].(map[string]interface{}); ok {
				if config["enabled"] != true {
					t.Errorf("getServiceValuesMap() merged config.enabled = %v, want %v", config["enabled"], true)
				}
			} else {
				t.Error("getServiceValuesMap() should preserve nested config structure")
			}
		}
	}
}

func TestGetServiceValuesMap_EmptySpec(t *testing.T) {
	rec := setupTestReconciler(t, true)
	handler, err := newTestHandler(t, WithTestReconciler(rec))
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	ctx := context.Background()
	services := []string{"service1"}
	result := handler.getServiceValuesMap(ctx, services, "default", "default")

	// Should return empty merged values when no CRD spec exists
	if service1Data, ok := result["service1"]; !ok {
		t.Error("getServiceValuesMap() should return data for service1 even with empty spec")
	} else {
		merged, ok := service1Data["merged"].(map[string]interface{})
		if !ok {
			t.Error("getServiceValuesMap() should return merged values as map")
		} else {
			if len(merged) != 0 {
				t.Errorf("getServiceValuesMap() should return empty merged map when no spec exists, got %v", merged)
			}
		}
	}
}

