package reconciler

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestReconciler_ResolveGVK(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	// Test with known kind
	gvk, err := rec.resolveGVK("Deployment")
	if err != nil {
		t.Fatalf("resolveGVK() error = %v", err)
	}

	if gvk.Kind != "Deployment" {
		t.Errorf("resolveGVK() Kind = %v, want Deployment", gvk.Kind)
	}

	if gvk.Group != "apps" {
		t.Errorf("resolveGVK() Group = %v, want apps", gvk.Group)
	}

	if gvk.Version != "v1" {
		t.Errorf("resolveGVK() Version = %v, want v1", gvk.Version)
	}

	// Test with unknown kind (should return generic)
	gvk, err = rec.resolveGVK("UnknownKind")
	if err != nil {
		t.Fatalf("resolveGVK() error = %v", err)
	}

	if gvk.Kind != "UnknownKind" {
		t.Errorf("resolveGVK() Kind = %v, want UnknownKind", gvk.Kind)
	}
}

func TestReconciler_ResolveResourceName(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	// Test with known GVK
	gvk := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}

	resource := rec.resolveResourceName(gvk)
	if resource != "deployments" {
		t.Errorf("resolveResourceName() = %v, want deployments", resource)
	}

	// Test with unknown GVK (should use pluralized kind)
	gvk = schema.GroupVersionKind{
		Kind: "CustomResource",
	}

	resource = rec.resolveResourceName(gvk)
	if resource != "customresources" {
		t.Errorf("resolveResourceName() = %v, want customresources", resource)
	}
}

func TestReconciler_GetObjectForGVK(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	gvk := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}

	obj := rec.getObjectForGVK(gvk, "default", "test-deployment")

	if obj.GetName() != "test-deployment" {
		t.Errorf("getObjectForGVK() name = %v, want test-deployment", obj.GetName())
	}

	if obj.GetNamespace() != "default" {
		t.Errorf("getObjectForGVK() namespace = %v, want default", obj.GetNamespace())
	}

	if obj.GroupVersionKind() != gvk {
		t.Errorf("getObjectForGVK() GVK = %v, want %v", obj.GroupVersionKind(), gvk)
	}
}

func TestReconciler_GetObjectForGVK_ClusterScoped(t *testing.T) {
	rec := setupTestReconcilerForTests(t)

	gvk := schema.GroupVersionKind{
		Kind: "Namespace",
	}

	obj := rec.getObjectForGVK(gvk, "", "test-namespace")

	if obj.GetName() != "test-namespace" {
		t.Errorf("getObjectForGVK() name = %v, want test-namespace", obj.GetName())
	}

	if obj.GetNamespace() != "" {
		t.Errorf("getObjectForGVK() namespace = %v, want empty", obj.GetNamespace())
	}
}
