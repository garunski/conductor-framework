package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func int32Ptr(i int32) *int32 {
	return &i
}

// GetServiceParameters retrieves parameters for a specific service from the spec
func (h *Handler) GetServiceParameters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := chi.URLParam(r, "service")
	
	// Get instance name from query parameter
	instanceName := getInstanceName(r)
	
	// Detect namespace from manifests
	detectedNamespace := h.getDetectedNamespace()
	if detectedNamespace == "" {
		detectedNamespace = "default"
	}

	if serviceName == "" {
		WriteErrorResponse(w, h.logger, http.StatusBadRequest, "invalid_request", "service name is required", nil)
		return
	}

	spec, err := h.parameterClient.GetSpec(ctx, instanceName, detectedNamespace)
	if err != nil || spec == nil || len(spec) == 0 {
		// Fallback to default namespace
		if detectedNamespace != "default" {
			spec, err = h.parameterClient.GetSpec(ctx, instanceName, "default")
		}
		if err != nil {
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "get_service_parameters_failed", err.Error(), nil)
			return
		}
	}

	// Extract service from spec
	var serviceParams interface{}
	if spec != nil {
		services, ok := spec["services"].(map[string]interface{})
		if ok {
			serviceParams = services[serviceName]
		}
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, serviceParams)
}
