package api

import (
	"testing"
)

func TestGetDetectedNamespace(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	// Add manifests in different namespaces
	testManifest1 := createTestManifest("Service", "test1", "default")
	testManifest2 := createTestManifest("Service", "test2", "default")
	testManifest3 := createTestManifest("Service", "test3", "test-ns")
	if err := handler.store.Create("default/Service/test1", []byte(testManifest1)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}
	if err := handler.store.Create("default/Service/test2", []byte(testManifest2)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}
	if err := handler.store.Create("test-ns/Service/test3", []byte(testManifest3)); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	detected := handler.getDetectedNamespace()
	if detected != "default" {
		t.Errorf("getDetectedNamespace() = %v, want default", detected)
	}
}

func TestMergeDescriptions(t *testing.T) {
	clusterSchema := map[string]interface{}{
		"properties": map[string]interface{}{
			"field1": map[string]interface{}{
				"type": "string",
			},
			"field2": map[string]interface{}{
				"type": "number",
			},
		},
	}

	sampleSchema := map[string]interface{}{
		"description": "root description",
		"properties": map[string]interface{}{
			"field1": map[string]interface{}{
				"description": "field1 description",
			},
			"field2": map[string]interface{}{
				"description": "field2 description",
			},
		},
	}

	mergeDescriptions(clusterSchema, sampleSchema)

	// Check root description
	if clusterSchema["description"] != "root description" {
		t.Error("mergeDescriptions() did not merge root description")
	}

	// Check field descriptions
	props := clusterSchema["properties"].(map[string]interface{})
	field1 := props["field1"].(map[string]interface{})
	field2 := props["field2"].(map[string]interface{})

	if field1["description"] != "field1 description" {
		t.Error("mergeDescriptions() did not merge field1 description")
	}
	if field2["description"] != "field2 description" {
		t.Error("mergeDescriptions() did not merge field2 description")
	}
}

func TestInt32Ptr(t *testing.T) {
	val := int32(42)
	ptr := int32Ptr(val)
	if ptr == nil {
		t.Error("int32Ptr() returned nil")
	}
	if *ptr != val {
		t.Errorf("int32Ptr() = %v, want %v", *ptr, val)
	}
}

