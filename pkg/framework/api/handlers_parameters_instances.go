package api

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

// ListParameterInstances lists all parameter instances in the namespace
func (h *Handler) ListParameterInstances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get namespace (instance not needed for listing)
	detectedNamespace, _ := h.getNamespaceAndInstance(r)
	
	// List all instances
	instances, err := h.parameterClient.List(ctx, detectedNamespace)
	if err != nil {
		// If not found in detected namespace, try default namespace
		if detectedNamespace != "default" {
			instances, err = h.parameterClient.List(ctx, "default")
		}
		if err != nil {
			// If listing fails (e.g., no Kubernetes cluster), return at least "default"
			// This allows the UI to work even when cluster is unavailable
			h.logger.V(1).Info("failed to list parameter instances, returning default only", "error", err)
			WriteJSONResponse(w, h.logger, http.StatusOK, []string{"default"})
			return
		}
	}
	
	// Extract instance names
	instanceNames := make([]string, 0, len(instances))
	for _, instance := range instances {
		instanceNames = append(instanceNames, instance.Name)
	}
	
	// If no instances found, ensure "default" is always available
	if len(instanceNames) == 0 {
		instanceNames = []string{"default"}
	} else {
		// Check if "default" is in the list, if not add it
		hasDefault := false
		for _, name := range instanceNames {
			if name == crd.DefaultName {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			instanceNames = append(instanceNames, crd.DefaultName)
		}
	}
	
	// Sort for consistent ordering
	sort.Strings(instanceNames)
	
	WriteJSONResponse(w, h.logger, http.StatusOK, instanceNames)
}

// CreateParameterInstance creates a new parameter instance with auto-generated name
func (h *Handler) CreateParameterInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get namespace (instance not needed for listing)
	detectedNamespace, _ := h.getNamespaceAndInstance(r)
	
	// List existing instances to find next available name
	existingInstances, err := h.parameterClient.List(ctx, detectedNamespace)
	if err != nil {
		// If not found in detected namespace, try default namespace
		if detectedNamespace != "default" {
			existingInstances, err = h.parameterClient.List(ctx, "default")
			if err == nil {
				detectedNamespace = "default"
			}
		}
		if err != nil {
			h.logger.Error(err, "failed to list existing instances, starting with config-1")
			existingInstances = []crd.DeploymentParameters{}
		}
	}
	
	// Build set of existing instance names
	existingNames := make(map[string]bool)
	for _, instance := range existingInstances {
		existingNames[instance.Name] = true
	}
	
	// Generate next available name (config-1, config-2, etc.)
	newInstanceName := ""
	for i := 1; i <= 1000; i++ {
		candidateName := fmt.Sprintf("config-%d", i)
		if !existingNames[candidateName] {
			newInstanceName = candidateName
			break
		}
	}
	
	if newInstanceName == "" {
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "name_generation_failed", "Could not generate a unique instance name", nil)
		return
	}
	
	// Validate name follows Kubernetes resource name rules
	if !isValidKubernetesName(newInstanceName) {
		WriteErrorResponse(w, h.logger, http.StatusBadRequest, "invalid_name", "Generated name does not follow Kubernetes naming rules", nil)
		return
	}
	
	// Try to copy from "default" instance if it exists, otherwise create empty
	var spec map[string]interface{}
	defaultInstance, err := h.parameterClient.Get(ctx, crd.DefaultName, detectedNamespace)
	if err == nil && defaultInstance != nil && defaultInstance.Spec != nil {
		// Deep copy the spec from default
		spec = deepCopySpecMap(defaultInstance.Spec)
	} else {
		// Create empty spec with default structure
		spec = map[string]interface{}{
			"global": map[string]interface{}{
				"namespace":  detectedNamespace,
				"namePrefix": "",
				"replicas":   int32(1),
			},
			"services": make(map[string]interface{}),
		}
	}
	
	// Create the new instance
	if err := h.parameterClient.CreateWithSpec(ctx, newInstanceName, detectedNamespace, spec); err != nil {
		h.logger.Error(err, "failed to create parameter instance", "name", newInstanceName)
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "create_instance_failed", err.Error(), nil)
		return
	}
	
	WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{
		"name":      newInstanceName,
		"namespace": detectedNamespace,
		"message":   fmt.Sprintf("Created parameter instance: %s", newInstanceName),
	})
}

// isValidKubernetesName validates that a name follows Kubernetes resource naming rules
// Names must be lowercase alphanumeric characters or '-', and must start and end with alphanumeric
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}
	
	for i, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
		// Must start and end with alphanumeric
		if (i == 0 || i == len(name)-1) && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	
	return true
}

