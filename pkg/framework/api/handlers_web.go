package api

import (
	"context"
	"net/http"
	"strings"
	"time"
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
	
	data := map[string]interface{}{
		"Services":           services,
		"InstallationStatus": installationStatus,
		// AppName and AppVersion will be added by renderTemplate
	}
	
	if err := h.renderTemplate(w, "deployments", data); err != nil {
		h.logger.Error(err, "failed to render template")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "template_execution_failed", "Failed to execute template", nil)
	}
}

// getServiceNames extracts service names from manifest directory structure
func (h *Handler) getServiceNames() []string {
	// Extract service names from embedded manifest files
	// Services are organized in directories: manifests/{service-name}/*.yaml
	serviceMap := make(map[string]bool)
	
	manifests := h.store.List()
	for key := range manifests {
		// Key format: namespace/kind/name
		// Extract service name from resource name
		// Services typically have names like: redis, postgresql, medusa-backend, etc.
		parts := strings.Split(key, "/")
		if len(parts) >= 3 {
			name := parts[2]
			
			// Map resource names to service names
			// Remove common suffixes and prefixes
			serviceName := name
			if strings.HasSuffix(name, "-backend") {
				serviceName = strings.TrimSuffix(name, "-backend")
			} else if strings.HasSuffix(name, "-pvc") {
				serviceName = strings.TrimSuffix(name, "-pvc")
			} else if strings.HasSuffix(name, "-secrets") {
				serviceName = strings.TrimSuffix(name, "-secrets")
			} else if strings.HasSuffix(name, "-config") {
				serviceName = strings.TrimSuffix(name, "-config")
			}
			
			// Known service names from manifest directories
			knownServices := []string{"redis", "postgresql", "medusa", "mercurjs", "uptrace", "clickhouse"}
			for _, knownSvc := range knownServices {
				if strings.HasPrefix(serviceName, knownSvc) || serviceName == knownSvc {
					serviceMap[knownSvc] = true
					break
				}
			}
		}
	}
	
	// Convert map to sorted slice
	services := make([]string, 0, len(serviceMap))
	for svc := range serviceMap {
		services = append(services, svc)
	}
	
	// If no services found, return default list
	if len(services) == 0 {
		return []string{"redis", "postgresql", "medusa", "mercurjs", "uptrace", "clickhouse"}
	}
	
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
