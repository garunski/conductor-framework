package reconciler

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Test deleteObject with existing resource
// Note: We test the delete logic even if fake client has limitations
func TestReconciler_deleteObject_Existing(t *testing.T) {
	rec := setupTestReconcilerForTests(t)
	ctx := context.Background()

	// Create resource using Create (which fake client supports better)
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-configmap",
				"namespace": "default",
			},
		},
	}

	key := "default/ConfigMap/test-configmap"
	// Use Create instead of Apply for fake client compatibility
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	_, err := rec.dynamicClient.Resource(gvr).Namespace("default").Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create resource for deletion test: %v", err)
	}

	// Now delete it
	err = rec.deleteObject(ctx, obj, key)
	if err != nil {
		t.Fatalf("deleteObject() error = %v", err)
	}

	// Verify resource was deleted
	_, err = rec.dynamicClient.Resource(gvr).Namespace("default").Get(ctx, "test-configmap", metav1.GetOptions{})
	if err == nil {
		t.Error("deleteObject() resource still exists after deletion")
	}
}

// Test deleteObject with non-existent resource (should handle gracefully)
func TestReconciler_deleteObject_NotFound(t *testing.T) {
	rec := setupTestReconcilerForTests(t)
	ctx := context.Background()

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "non-existent",
				"namespace": "default",
			},
		},
	}

	key := "default/ConfigMap/non-existent"
	err := rec.deleteObject(ctx, obj, key)
	// Should not error for not found
	if err != nil {
		t.Errorf("deleteObject() error = %v, expected nil for not found resource", err)
	}
}

// Test deleteObject with missing kind
func TestReconciler_deleteObject_MissingKind(t *testing.T) {
	rec := setupTestReconcilerForTests(t)
	ctx := context.Background()

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"metadata": map[string]interface{}{
				"name": "test-resource",
			},
		},
	}

	key := "default/Resource/test-resource"
	err := rec.deleteObject(ctx, obj, key)
	if err == nil {
		t.Error("deleteObject() expected error for missing kind, got nil")
	}
}
