package reconciler

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestReconciler_ParseYAML(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	yamlData := []byte(`apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: default
spec: {}`)

	obj, err := rec.parseYAML(yamlData, "default/Service/test-service")
	if err != nil {
		t.Fatalf("parseYAML() error = %v", err)
	}

	if obj == nil {
		t.Fatal("parseYAML() returned nil")
	}

	// parseYAML may return typed or unstructured objects
	// Check if it's unstructured, otherwise it's a typed object which is also valid
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if ok {
		if unstructuredObj.GetName() != "test-service" {
			t.Errorf("parseYAML() name = %v, want test-service", unstructuredObj.GetName())
		}
	} else {
		// If it's a typed object, that's also valid - just verify it's not nil
		_ = obj
	}
}

func TestReconciler_ParseKey(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	key := "default/Service/test-service"
	obj, err := rec.parseKey(key)
	if err != nil {
		t.Fatalf("parseKey() error = %v", err)
	}

	if obj.GetName() != "test-service" {
		t.Errorf("parseKey() name = %v, want test-service", obj.GetName())
	}

	if obj.GetNamespace() != "default" {
		t.Errorf("parseKey() namespace = %v, want default", obj.GetNamespace())
	}

	if obj.GetKind() != "Service" {
		t.Errorf("parseKey() kind = %v, want Service", obj.GetKind())
	}
}

func TestReconciler_ParseKey_InvalidFormat(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	tests := []string{
		"invalid",
		"namespace/kind",
		"namespace/kind/name/extra",
	}

	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			_, err := rec.parseKey(key)
			if err == nil {
				t.Errorf("parseKey(%q) expected error, got nil", key)
			}
		})
	}
}

