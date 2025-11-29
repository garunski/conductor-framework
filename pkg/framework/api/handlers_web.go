package api

import (
	"context"
	"net/http"
	"sort"
	"time"

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
