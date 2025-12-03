package api

import (
	"fmt"
	"net/http"
	"strings"
)

func (h *Handler) Up(w http.ResponseWriter, r *http.Request) {
	if h.reconciler == nil {
		WriteErrorResponse(w, h.logger, http.StatusServiceUnavailable, "reconciler_unavailable", "Reconciler not available", nil)
		return
	}

	ctx := r.Context()
	
	// Parse request body for service selection
	var req DeploymentRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := h.parseJSONRequest(r, &req); err != nil {
			WriteErrorResponse(w, h.logger, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
	}
	
	manifests := h.store.List()
	
	// If services are specified, filter manifests
	if len(req.Services) > 0 {
		manifests = filterManifestsByServices(manifests, req.Services)
		if len(manifests) == 0 {
			WriteErrorResponse(w, h.logger, http.StatusBadRequest, "no_manifests", "No manifests found for selected services", nil)
			return
		}
	}
	
	// Get instance name from query parameter
	instanceName := getInstanceName(r)
	
	// Re-render manifests with current parameters before deploying
	updatedManifests, err := h.updateManifestsWithCurrentParameters(ctx, manifests, instanceName)
	if err != nil {
		h.logger.Error(err, "failed to update manifests with current parameters, using existing manifests")
	} else {
		manifests = updatedManifests
	}
	
	if len(req.Services) > 0 {
		if err := h.reconciler.DeployManifests(ctx, manifests); err != nil {
			h.logger.Error(err, "failed to deploy selected services")
			serviceList := strings.Join(req.Services, ", ")
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "deployment_failed", fmt.Sprintf("Deployment failed for service(s): %s. Error: %s", serviceList, err.Error()), nil)
			return
		}
		
		serviceList := strings.Join(req.Services, ", ")
		WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{
			"message": fmt.Sprintf("Deployment initiated for %d service(s): %s", len(req.Services), serviceList),
		})
		return
	}
	
	// No services specified, deploy all using updated manifests with current namespace from CRD
	if err := h.reconciler.DeployManifests(ctx, manifests); err != nil {
		h.logger.Error(err, "failed to deploy all")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "deployment_failed", fmt.Sprintf("Deployment failed for all services. Error: %s", err.Error()), nil)
		return
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{"message": "Deployment initiated for all services"})
}

func (h *Handler) Down(w http.ResponseWriter, r *http.Request) {
	if h.reconciler == nil {
		WriteErrorResponse(w, h.logger, http.StatusServiceUnavailable, "reconciler_unavailable", "Reconciler not available", nil)
		return
	}

	ctx := r.Context()
	
	// Parse request body for service selection
	var req DeploymentRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := h.parseJSONRequest(r, &req); err != nil {
			WriteErrorResponse(w, h.logger, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
	}
	
	manifests := h.store.List()
	
	// If services are specified, filter manifests
	if len(req.Services) > 0 {
		manifests = filterManifestsByServices(manifests, req.Services)
		if len(manifests) == 0 {
			WriteErrorResponse(w, h.logger, http.StatusBadRequest, "no_manifests", "No manifests found for selected services", nil)
			return
		}
		
		if err := h.reconciler.DeleteManifests(ctx, manifests); err != nil {
			h.logger.Error(err, "failed to delete selected services")
			serviceList := strings.Join(req.Services, ", ")
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "deletion_failed", fmt.Sprintf("Deletion failed for service(s): %s. Error: %s", serviceList, err.Error()), nil)
			return
		}
		
		serviceList := strings.Join(req.Services, ", ")
		WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{
			"message": fmt.Sprintf("Deletion completed for %d service(s): %s", len(req.Services), serviceList),
		})
		return
	}
	
	// No services specified, delete all
	if err := h.reconciler.DeleteAll(ctx); err != nil {
		h.logger.Error(err, "failed to delete all")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "deletion_failed", fmt.Sprintf("Deletion failed for all services. Error: %s", err.Error()), nil)
		return
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{"message": "Deletion completed for all services"})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if h.reconciler == nil {
		WriteErrorResponse(w, h.logger, http.StatusServiceUnavailable, "reconciler_unavailable", "Reconciler not available", nil)
		return
	}

	ctx := r.Context()
	
	// Parse request body for service selection
	var req DeploymentRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := h.parseJSONRequest(r, &req); err != nil {
			WriteErrorResponse(w, h.logger, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
	}
	
	manifests := h.store.List()
	
	// If services are specified, filter manifests
	if len(req.Services) > 0 {
		manifests = filterManifestsByServices(manifests, req.Services)
		if len(manifests) == 0 {
			WriteErrorResponse(w, h.logger, http.StatusBadRequest, "no_manifests", "No manifests found for selected services", nil)
			return
		}
	}
	
	// Get instance name from query parameter
	instanceName := getInstanceName(r)
	
	// Re-render manifests with current parameters before updating
	updatedManifests, err := h.updateManifestsWithCurrentParameters(ctx, manifests, instanceName)
	if err != nil {
		h.logger.Error(err, "failed to update manifests with current parameters, using existing manifests")
	} else {
		manifests = updatedManifests
	}
	
	if len(req.Services) > 0 {
		if err := h.reconciler.UpdateManifests(ctx, manifests); err != nil {
			h.logger.Error(err, "failed to update selected services")
			serviceList := strings.Join(req.Services, ", ")
			WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "update_failed", fmt.Sprintf("Update failed for service(s): %s. Error: %s", serviceList, err.Error()), nil)
			return
		}
		
		serviceList := strings.Join(req.Services, ", ")
		WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{
			"message": fmt.Sprintf("Update initiated for %d service(s): %s", len(req.Services), serviceList),
		})
		return
	}
	
	// No services specified, update all using updated manifests with current namespace from CRD
	if err := h.reconciler.UpdateManifests(ctx, manifests); err != nil {
		h.logger.Error(err, "failed to update all")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "update_failed", fmt.Sprintf("Update failed for all services. Error: %s", err.Error()), nil)
		return
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{"message": "Update initiated for all services"})
}
