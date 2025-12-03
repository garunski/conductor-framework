package api

import (
	"net/http"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

// getInstanceName extracts the instance name from the query parameter, defaults to "default"
func getInstanceName(r *http.Request) string {
	instance := r.URL.Query().Get("instance")
	if instance == "" {
		return crd.DefaultName
	}
	return instance
}

// getDetectedNamespace extracts the most common namespace from manifests
func (h *Handler) getDetectedNamespace() string {
	manifests := h.store.List()
	return detectNamespaceFromManifests(manifests)
}

// GetParameters retrieves the current deployment parameters
func (h *Handler) GetParameters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get instance name from query parameter
	instanceName := getInstanceName(r)
	
	// Detect namespace from manifests
	detectedNamespace := h.getDetectedNamespace()
	if detectedNamespace == "" {
		detectedNamespace = "default"
	}

	spec, err := h.parameterClient.GetSpec(ctx, instanceName, detectedNamespace)
	if err != nil || spec == nil || len(spec) == 0 {
		// Fallback to default namespace
		if detectedNamespace != "default" {
			spec, err = h.parameterClient.GetSpec(ctx, crd.DefaultName, "default")
		}
		if err != nil {
			h.logger.Error(err, "failed to get DeploymentParameters spec")
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "get_parameters_failed", err.Error(), nil)
			return
		}
	}
	if err != nil {
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "get_parameters_failed", err.Error(), nil)
		return
	}

	if spec == nil || len(spec) == 0 {
		// Return default parameters if CRD doesn't exist
		spec = map[string]interface{}{
			"global": map[string]interface{}{
				"namespace":  "default",
				"namePrefix": "",
				"replicas":   int32(1),
			},
		}
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, spec)
}

// UpdateParameters creates or updates deployment parameters
func (h *Handler) UpdateParameters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get instance name from query parameter
	instanceName := getInstanceName(r)
	
	// Detect namespace from manifests
	detectedNamespace := h.getDetectedNamespace()
	if detectedNamespace == "" {
		detectedNamespace = "default"
	}

	var spec map[string]interface{}

	if err := h.parseJSONRequest(r, &spec); err != nil {
		WriteErrorResponse(w, h.logger, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	// Get existing parameters to check if it exists
	params, err := h.parameterClient.Get(ctx, instanceName, detectedNamespace)
	if err != nil {
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "get_parameters_failed", err.Error(), nil)
		return
	}

	if params == nil {
		// Create new
		if err := h.parameterClient.CreateWithSpec(ctx, instanceName, detectedNamespace, spec); err != nil {
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "create_parameters_failed", err.Error(), nil)
			return
		}
	} else {
		// Update existing
		if err := h.parameterClient.UpdateSpec(ctx, instanceName, detectedNamespace, spec); err != nil {
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "update_parameters_failed", err.Error(), nil)
			return
		}
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{"message": "Parameters updated successfully"})
}
