package api

// createTestManifest creates a test Kubernetes manifest YAML string
func createTestManifest(kind, name, namespace string) string {
	ns := ""
	if namespace != "" {
		ns = "  namespace: " + namespace + "\n"
	}
	return `apiVersion: v1
kind: ` + kind + `
metadata:
  name: ` + name + `
` + ns + `spec: {}
`
}

// mapsEqual compares two maps recursively for testing
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

