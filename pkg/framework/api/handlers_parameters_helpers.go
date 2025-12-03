package api

import (
	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

// deepCopySpecMap creates a deep copy of a DeploymentParametersSpec map
func deepCopySpecMap(src crd.DeploymentParametersSpec) map[string]interface{} {
	return deepCopyMapInterface(src)
}

// deepCopyMapInterface creates a deep copy of a map[string]interface{}
func deepCopyMapInterface(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[k] = deepCopyMapInterface(val)
		case []interface{}:
			dst[k] = deepCopySliceInterface(val)
		default:
			dst[k] = v
		}
	}
	return dst
}

// deepCopySliceInterface creates a deep copy of a []interface{}
func deepCopySliceInterface(src []interface{}) []interface{} {
	dst := make([]interface{}, len(src))
	for i, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[i] = deepCopyMapInterface(val)
		case []interface{}:
			dst[i] = deepCopySliceInterface(val)
		default:
			dst[i] = v
		}
	}
	return dst
}
