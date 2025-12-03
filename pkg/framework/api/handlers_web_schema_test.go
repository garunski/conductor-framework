package api

import (
	"testing"
)

func TestMergeSchemaWithInstance(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		instance map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "schema with instance values",
			schema: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": map[string]interface{}{},
					"replicas":  map[string]interface{}{},
				},
				"services": map[string]interface{}{},
			},
			instance: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": "default",
					"replicas":  float64(3),
				},
			},
			expected: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": "default",
					"replicas":  float64(3),
				},
				"services": map[string]interface{}{},
			},
		},
		{
			name: "empty schema with instance",
			schema: map[string]interface{}{
				"global":   map[string]interface{}{},
				"services": map[string]interface{}{},
			},
			instance: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": "test",
				},
				"services": map[string]interface{}{
					"service1": map[string]interface{}{
						"replicas": float64(2),
					},
				},
			},
			expected: map[string]interface{}{
				"global": map[string]interface{}{
					"namespace": "test",
				},
				"services": map[string]interface{}{
					"service1": map[string]interface{}{
						"replicas": float64(2),
					},
				},
			},
		},
		{
			name: "nested schema with instance",
			schema: map[string]interface{}{
				"services": map[string]interface{}{
					"config": map[string]interface{}{
						"database": map[string]interface{}{
							"host": map[string]interface{}{},
						},
					},
				},
			},
			instance: map[string]interface{}{
				"services": map[string]interface{}{
					"api-service": map[string]interface{}{
						"config": map[string]interface{}{
							"database": map[string]interface{}{
								"host": "localhost",
								"port": float64(5432),
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"global": map[string]interface{}{},
				"services": map[string]interface{}{
					"api-service": map[string]interface{}{
						"config": map[string]interface{}{
							"database": map[string]interface{}{
								"host": "localhost",
								"port": float64(5432),
							},
						},
					},
				},
			},
		},
		{
			name: "schema template with multiple services",
			schema: map[string]interface{}{
				"services": map[string]interface{}{
					"replicas": map[string]interface{}{},
					"config":   map[string]interface{}{},
				},
			},
			instance: map[string]interface{}{
				"services": map[string]interface{}{
					"service1": map[string]interface{}{
						"replicas": float64(3),
					},
					"service2": map[string]interface{}{
						"replicas": float64(2),
						"config": map[string]interface{}{
							"enabled": true,
						},
					},
				},
			},
			expected: map[string]interface{}{
				"global": map[string]interface{}{},
				"services": map[string]interface{}{
					"service1": map[string]interface{}{
						"replicas": float64(3),
						"config":   map[string]interface{}{},
					},
					"service2": map[string]interface{}{
						"replicas": float64(2),
						"config": map[string]interface{}{
							"enabled": true,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeSchemaWithInstance(tt.schema, tt.instance)
			
			// Verify global
			if resultGlobal, ok := result["global"].(map[string]interface{}); ok {
				if expectedGlobal, ok := tt.expected["global"].(map[string]interface{}); ok {
					if !mapsEqual(resultGlobal, expectedGlobal) {
						t.Errorf("mergeSchemaWithInstance() global = %v, want %v", resultGlobal, expectedGlobal)
					}
				}
			}
			
			// Verify services
			if resultServices, ok := result["services"].(map[string]interface{}); ok {
				if expectedServices, ok := tt.expected["services"].(map[string]interface{}); ok {
					if !mapsEqual(resultServices, expectedServices) {
						t.Errorf("mergeSchemaWithInstance() services = %v, want %v", resultServices, expectedServices)
					}
				}
			}
		})
	}
}

