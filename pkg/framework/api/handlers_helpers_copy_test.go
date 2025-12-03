package api

import (
	"testing"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

func TestDeepCopySpecMap(t *testing.T) {
	src := crd.DeploymentParametersSpec{
		"global": map[string]interface{}{
			"namespace": "test",
			"replicas":  int32(3),
		},
		"services": map[string]interface{}{
			"service1": map[string]interface{}{
				"replicas": int32(2),
			},
		},
	}

	dst := deepCopySpecMap(src)

	// Modify original
	src["global"].(map[string]interface{})["namespace"] = "modified"

	// Check that copy wasn't affected
	if dst["global"].(map[string]interface{})["namespace"] != "test" {
		t.Error("deepCopySpecMap() did not create a deep copy")
	}
}

func TestDeepCopyMapInterface(t *testing.T) {
	src := map[string]interface{}{
		"nested": map[string]interface{}{
			"value": "test",
		},
		"slice": []interface{}{1, 2, 3},
	}

	dst := deepCopyMapInterface(src)

	// Modify original
	src["nested"].(map[string]interface{})["value"] = "modified"
	src["slice"].([]interface{})[0] = 999

	// Check that copy wasn't affected
	if dst["nested"].(map[string]interface{})["value"] != "test" {
		t.Error("deepCopyMapInterface() did not create a deep copy of nested map")
	}
	if dst["slice"].([]interface{})[0] != 1 {
		t.Error("deepCopyMapInterface() did not create a deep copy of slice")
	}
}

func TestDeepCopySliceInterface(t *testing.T) {
	src := []interface{}{
		map[string]interface{}{"value": "test"},
		[]interface{}{1, 2},
		"string",
	}

	dst := deepCopySliceInterface(src)

	// Modify original
	src[0].(map[string]interface{})["value"] = "modified"
	src[1].([]interface{})[0] = 999

	// Check that copy wasn't affected
	if dst[0].(map[string]interface{})["value"] != "test" {
		t.Error("deepCopySliceInterface() did not create a deep copy of nested map")
	}
	if dst[1].([]interface{})[0] != 1 {
		t.Error("deepCopySliceInterface() did not create a deep copy of nested slice")
	}
	if dst[2] != "string" {
		t.Error("deepCopySliceInterface() did not copy string value")
	}
}

