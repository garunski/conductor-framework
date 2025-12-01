package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
	"gopkg.in/yaml.v3"
)

func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	// Get list of services from manifest directories
	services := h.getServiceNames()
	
	// Get installation status for each service
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	manifests := h.store.List()
	installationStatus := getServiceInstallationStatus(ctx, services, manifests, h.reconciler)
	
	data := map[string]interface{}{
		"Services":           services,
		"InstallationStatus": installationStatus,
		// AppName and AppVersion will be added by renderTemplate
	}
	
	if err := h.renderTemplate(w, "service-health-page", data); err != nil {
		h.logger.Error(err, "failed to render template")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "template_execution_failed", "Failed to execute template", nil)
	}
}

func (h *Handler) DeploymentsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	// Get list of services from manifest directories
	services := h.getServiceNames()
	
	// Get installation status for each service
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	manifests := h.store.List()
	installationStatus := getServiceInstallationStatus(ctx, services, manifests, h.reconciler)
	
	// Detect namespace from manifests (try to find the most common namespace)
	detectedNamespace := detectNamespaceFromManifests(manifests)
	if detectedNamespace == "" {
		detectedNamespace = "default"
	}
	
	// Get CRD schema definition (raw OpenAPI schema) for form generation
	crdSchema, err := h.parameterClient.GetCRDSchema(ctx)
	if err != nil {
		h.logger.V(1).Info("failed to get CRD schema, using sample schema for local development", "error", err)
		// Use sample schema for local development/debugging
		crdSchema = GetSampleCRDSchema()
	}
	
	// Extract the spec schema from the CRD schema
	var specSchema map[string]interface{}
	if properties, ok := crdSchema["properties"].(map[string]interface{}); ok {
		if spec, ok := properties["spec"].(map[string]interface{}); ok {
			specSchema = spec
		}
	}
	
	// If we still don't have a spec schema, use the sample one
	if specSchema == nil || len(specSchema) == 0 {
		sampleSchema := GetSampleCRDSchema()
		if properties, ok := sampleSchema["properties"].(map[string]interface{}); ok {
			if spec, ok := properties["spec"].(map[string]interface{}); ok {
				specSchema = spec
			}
		}
	}
	
	// Get CRD instance values - try detected namespace first, then fallback to default
	instanceSpec, err := h.parameterClient.GetSpec(ctx, crd.DefaultName, detectedNamespace)
	if err != nil || instanceSpec == nil || len(instanceSpec) == 0 {
		// Fallback to default namespace if not found in detected namespace
		if detectedNamespace != "default" {
			instanceSpec, err = h.parameterClient.GetSpec(ctx, crd.DefaultName, "default")
		}
		if err != nil || instanceSpec == nil {
			instanceSpec = make(map[string]interface{})
		}
	}
	
	// Ensure services map exists to avoid nil index errors in template
	if instanceSpec["services"] == nil {
		instanceSpec["services"] = make(map[string]interface{})
	}
	
	// Get service values for current values display
	serviceValues := h.getServiceValuesMap(ctx, services, detectedNamespace)
	
	// Convert schema and instance to JSON for JavaScript library
	var specSchemaJSON, instanceSpecJSON string
	if specSchema != nil {
		if b, err := json.Marshal(specSchema); err == nil {
			specSchemaJSON = string(b)
		}
	}
	if instanceSpec != nil {
		if b, err := json.Marshal(instanceSpec); err == nil {
			instanceSpecJSON = string(b)
		}
	}
	
	data := map[string]interface{}{
		"Services":           services,
		"InstallationStatus": installationStatus,
		"ParametersSpec":     instanceSpec, // Keep for backward compatibility
		"ServiceValues":      serviceValues,
		"CRDSchemaJSON":      specSchemaJSON, // Raw JSON schema for JavaScript library
		"InstanceSpecJSON":   instanceSpecJSON, // Instance values as JSON
		// AppName and AppVersion will be added by renderTemplate
	}
	
	if err := h.renderTemplate(w, "deployments", data); err != nil {
		h.logger.Error(err, "failed to render template")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "template_execution_failed", "Failed to execute template", nil)
	}
}

func (h *Handler) ParametersPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	// Get list of services from manifest directories
	services := h.getServiceNames()
	
	// Detect namespace from manifests (try to find the most common namespace)
	manifests := h.store.List()
	detectedNamespace := detectNamespaceFromManifests(manifests)
	if detectedNamespace == "" {
		detectedNamespace = "default"
	}
	
	// Get CRD schema definition (raw OpenAPI schema) for form generation
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	crdSchema, err := h.parameterClient.GetCRDSchema(ctx)
	if err != nil {
		h.logger.V(1).Info("failed to get CRD schema, using sample schema for local development", "error", err)
		crdSchema = GetSampleCRDSchema()
	}
	
	// Extract the spec schema from the CRD schema
	var specSchema map[string]interface{}
	if properties, ok := crdSchema["properties"].(map[string]interface{}); ok {
		if spec, ok := properties["spec"].(map[string]interface{}); ok {
			specSchema = spec
		}
	}
	
	// If we still don't have a spec schema, use the sample one
	if specSchema == nil || len(specSchema) == 0 {
		sampleSchema := GetSampleCRDSchema()
		if properties, ok := sampleSchema["properties"].(map[string]interface{}); ok {
			if spec, ok := properties["spec"].(map[string]interface{}); ok {
				specSchema = spec
			}
		}
	}
	
	// Get CRD instance values - try detected namespace first, then fallback to default
	instanceSpec, err := h.parameterClient.GetSpec(ctx, crd.DefaultName, detectedNamespace)
	if err != nil || instanceSpec == nil || len(instanceSpec) == 0 {
		// Fallback to default namespace if not found in detected namespace
		if detectedNamespace != "default" {
			instanceSpec, err = h.parameterClient.GetSpec(ctx, crd.DefaultName, "default")
		}
		if err != nil || instanceSpec == nil {
			instanceSpec = make(map[string]interface{})
		}
	}
	
	// Ensure services map exists to avoid nil index errors in template
	if instanceSpec["services"] == nil {
		instanceSpec["services"] = make(map[string]interface{})
	}
	
	// Convert schema and instance to JSON for JavaScript library
	var specSchemaJSON, instanceSpecJSON string
	if specSchema != nil {
		if b, err := json.Marshal(specSchema); err == nil {
			specSchemaJSON = string(b)
		}
	}
	if instanceSpec != nil {
		if b, err := json.Marshal(instanceSpec); err == nil {
			instanceSpecJSON = string(b)
		}
	}
	
	data := map[string]interface{}{
		"Services":           services,
		"ParametersSpec":     instanceSpec, // Keep for backward compatibility
		"CRDSchemaJSON":      specSchemaJSON, // Raw JSON schema for JavaScript library
		"InstanceSpecJSON":   instanceSpecJSON, // Instance values as JSON
		// AppName and AppVersion will be added by renderTemplate
	}
	
	if err := h.renderTemplate(w, "parameters", data); err != nil {
		h.logger.Error(err, "failed to render template")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "template_execution_failed", "Failed to execute template", nil)
	}
}

// getServiceNames extracts service names from manifest directory structure
// It discovers services by looking at Service resources in the manifests
func (h *Handler) getServiceNames() []string {
	// Extract service names from Service resources in manifests
	// Services are organized in directories: manifests/{service-name}/*.yaml
	serviceMap := make(map[string]bool)
	
	manifests := h.store.List()
	for _, yamlData := range manifests {
		// Parse YAML to check if it's a Service
		var obj struct {
			Kind     string `yaml:"kind"`
			Metadata struct {
				Name string `yaml:"name"`
			} `yaml:"metadata"`
		}
		
		if err := yaml.Unmarshal(yamlData, &obj); err != nil {
			continue
		}
		
		// Only consider Service resources
		if obj.Kind != "Service" {
			continue
		}
		
		if obj.Metadata.Name == "" {
			continue
		}
		
		// Extract service name from Service resource name
		// For guestbook: frontend, redis-master, redis-slave
		// Use the service name as-is (may include name prefix if set in templates)
		serviceName := obj.Metadata.Name
		serviceMap[serviceName] = true
	}
	
	// Convert map to sorted slice
	services := make([]string, 0, len(serviceMap))
	for svc := range serviceMap {
		services = append(services, svc)
	}
	
	// Sort for consistent ordering
	sort.Strings(services)
	
	return services
}

func (h *Handler) LogsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := h.renderTemplate(w, "logs", nil); err != nil {
		h.logger.Error(err, "failed to render template")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "template_execution_failed", "Failed to execute template", nil)
	}
}

// getServiceValuesMap returns merged/default and deployed values for all services
// Similar to GetServiceValues but returns a map instead of writing HTTP response
func (h *Handler) getServiceValuesMap(ctx context.Context, services []string, defaultNamespace string) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})
	clientset := h.reconciler.GetClientset()
	manifests := h.store.List()

	for _, serviceName := range services {
		serviceData := make(map[string]interface{})

		// Get merged/default values
		var merged map[string]interface{}

		// Try to get spec, but don't fail if cluster is unavailable
		spec, err := h.parameterClient.GetSpec(ctx, crd.DefaultName, defaultNamespace)
		if err == nil && spec != nil {
			// Merge global and service-specific parameters
			merged = make(map[string]interface{})

			// Start with global defaults
			if global, ok := spec["global"].(map[string]interface{}); ok {
				for k, v := range global {
					merged[k] = v
				}
			}

			// Apply service-specific overrides
			if services, ok := spec["services"].(map[string]interface{}); ok {
				if service, ok := services[serviceName].(map[string]interface{}); ok {
					for k, v := range service {
						merged[k] = v
					}
				}
			}
		}

		// Fall back to empty map if no merged params
		if merged == nil || len(merged) == 0 {
			merged = make(map[string]interface{})
		}

		serviceData["merged"] = merged

		// Get actual deployed values from Kubernetes (only if cluster is available)
		if clientset != nil {
			deployCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			deployed := getDeployedValues(deployCtx, clientset, serviceName, defaultNamespace, manifests)
			cancel()
			if deployed != nil {
				serviceData["deployed"] = deployed
			}
		}

		result[serviceName] = serviceData
	}

	return result
}

// detectNamespaceFromManifests extracts the most common namespace from manifest keys
// Manifest keys are in format "namespace/Kind/name"
func detectNamespaceFromManifests(manifests map[string][]byte) string {
	if len(manifests) == 0 {
		return ""
	}
	
	namespaceCounts := make(map[string]int)
	for key := range manifests {
		parts := strings.Split(key, "/")
		if len(parts) >= 1 && parts[0] != "" {
			namespaceCounts[parts[0]]++
		}
	}
	
	// Find the most common namespace
	maxCount := 0
	mostCommon := ""
	for ns, count := range namespaceCounts {
		if count > maxCount {
			maxCount = count
			mostCommon = ns
		}
	}
	
	return mostCommon
}

// ServeStatic serves static files from the embedded filesystem
func (h *Handler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	// Get the file path from the URL
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	
	// Construct the full path in the embedded filesystem
	fullPath := filepath.Join("templates/static", path)
	
	// Try to read the file from embedded filesystem
	file, err := templateFiles.Open(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	
	// Set content type based on file extension
	ext := filepath.Ext(path)
	switch ext {
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".html":
		w.Header().Set("Content-Type", "text/html")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	
	// Set cache headers - no cache for JS files to prevent stale code
	if ext == ".js" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	
	// Read and serve the file content
	_, err = io.Copy(w, file)
	if err != nil {
		h.logger.Error(err, "failed to serve static file", "path", path)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// GetSampleCRDSchema returns a sample CRD schema for local development/debugging
// This is used when the real CRD cannot be fetched from the cluster
func GetSampleCRDSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"spec": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"global": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"namespace": map[string]interface{}{
								"type":    "string",
								"default": "default",
							},
							"namePrefix": map[string]interface{}{
								"type":    "string",
								"default": "",
							},
							"replicas": map[string]interface{}{
								"type":    "integer",
								"default": 1,
							},
							"imageTag": map[string]interface{}{
								"type": "string",
							},
							"imageRegistry": map[string]interface{}{
								"type":    "string",
								"default": "",
							},
							"imagePullSecrets": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"name": map[string]interface{}{
											"type": "string",
										},
									},
								},
							},
							"storageClassName": map[string]interface{}{
								"type":    "string",
								"default": "local-path",
							},
							"keepPVC": map[string]interface{}{
								"type":    "boolean",
								"default": false,
							},
							"resources": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"requests": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"memory": map[string]interface{}{
												"type": "string",
											},
											"cpu": map[string]interface{}{
												"type": "string",
											},
										},
									},
									"limits": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"memory": map[string]interface{}{
												"type": "string",
											},
											"cpu": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
							"storageSize": map[string]interface{}{
								"type": "string",
							},
							"labels": map[string]interface{}{
								"type": "object",
								"additionalProperties": map[string]interface{}{
									"type": "string",
								},
							},
							"annotations": map[string]interface{}{
								"type": "object",
								"additionalProperties": map[string]interface{}{
									"type": "string",
								},
							},
							"nodeSelector": map[string]interface{}{
								"type": "object",
								"additionalProperties": map[string]interface{}{
									"type": "string",
								},
							},
							"tolerations": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "object",
								},
							},
						},
					},
					"services": map[string]interface{}{
						"type": "object",
						"additionalProperties": map[string]interface{}{
							"type": "object",
							"additionalProperties": true,
							"properties": map[string]interface{}{
								"namespace": map[string]interface{}{
									"type": "string",
								},
								"namePrefix": map[string]interface{}{
									"type": "string",
								},
								"replicas": map[string]interface{}{
									"type": "integer",
								},
								"imageTag": map[string]interface{}{
									"type": "string",
								},
								"resources": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"requests": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"memory": map[string]interface{}{
													"type": "string",
												},
												"cpu": map[string]interface{}{
													"type": "string",
												},
											},
										},
										"limits": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"memory": map[string]interface{}{
													"type": "string",
												},
												"cpu": map[string]interface{}{
													"type": "string",
												},
											},
										},
									},
								},
								"storageSize": map[string]interface{}{
									"type": "string",
								},
								"labels": map[string]interface{}{
									"type": "object",
									"additionalProperties": map[string]interface{}{
										"type": "string",
									},
								},
								"annotations": map[string]interface{}{
									"type": "object",
									"additionalProperties": map[string]interface{}{
										"type": "string",
									},
								},
								// Example custom fields that might be in service configs
								"config": map[string]interface{}{
									"type": "object",
									"additionalProperties": true,
								},
								"ingress": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"enabled": map[string]interface{}{
											"type":    "boolean",
											"default": false,
										},
										"host": map[string]interface{}{
											"type": "string",
										},
										"path": map[string]interface{}{
											"type": "string",
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
}

// mergeSchemaWithInstance merges the CRD schema structure with instance values
// Schema provides the structure (what fields exist), instance provides the values
func mergeSchemaWithInstance(schema, instance map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Start with schema structure
	if schemaGlobal, ok := schema["global"].(map[string]interface{}); ok {
		result["global"] = deepCopyMap(schemaGlobal)
	} else {
		result["global"] = make(map[string]interface{})
	}
	
	// Get schema template for services (this is a template for what each service can have)
	var schemaServicesTemplate map[string]interface{}
	if schemaServices, ok := schema["services"].(map[string]interface{}); ok {
		schemaServicesTemplate = schemaServices
	}
	
	// Initialize services map
	result["services"] = make(map[string]interface{})
	
	// Overlay instance values on top of schema structure
	if instanceGlobal, ok := instance["global"].(map[string]interface{}); ok {
		mergeMaps(result["global"].(map[string]interface{}), instanceGlobal)
	}
	
	// For services, the schema structure is a template for each service
	// Merge schema template with each service's instance values
	if instanceServices, ok := instance["services"].(map[string]interface{}); ok {
		for serviceName, serviceInstance := range instanceServices {
			if serviceMap, ok := serviceInstance.(map[string]interface{}); ok {
				// Start with schema template if available
				var serviceResult map[string]interface{}
				if schemaServicesTemplate != nil {
					serviceResult = deepCopyMap(schemaServicesTemplate)
				} else {
					serviceResult = make(map[string]interface{})
				}
				// Merge instance values into the schema template
				mergeMaps(serviceResult, serviceMap)
				// Store the merged result
				result["services"].(map[string]interface{})[serviceName] = serviceResult
			} else {
				// If not a map, just use the instance value
				result["services"].(map[string]interface{})[serviceName] = serviceInstance
			}
		}
	}
	
	return result
}

// deepCopyMap creates a deep copy of a map
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		if vMap, ok := v.(map[string]interface{}); ok {
			result[k] = deepCopyMap(vMap)
		} else {
			result[k] = v
		}
	}
	return result
}

// mergeMaps merges source into destination, recursively
func mergeMaps(dest, src map[string]interface{}) {
	for k, v := range src {
		if vMap, ok := v.(map[string]interface{}); ok {
			if destMap, ok := dest[k].(map[string]interface{}); ok {
				mergeMaps(destMap, vMap)
			} else {
				dest[k] = deepCopyMap(vMap)
			}
		} else {
			dest[k] = v
		}
	}
}
