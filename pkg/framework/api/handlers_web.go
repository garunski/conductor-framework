package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
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
	
	// Get namespace and instance name
	detectedNamespace, instanceName := h.getNamespaceAndInstance(r)
	
	// Get list of services from manifest directories
	services := h.getServiceNames()
	
	// Get installation status for each service
	ctx, cancel := context.WithTimeout(r.Context(), DefaultRequestTimeout)
	defer cancel()
	
	manifests := h.store.List()
	installationStatus := getServiceInstallationStatus(ctx, services, manifests, h.reconciler)
	
	// Get CRD schema with fallback
	specSchema, err := h.getCRDSchemaWithFallback(ctx)
	if err != nil {
		h.logger.V(1).Info("failed to get CRD schema, using empty schema", "error", err)
		specSchema = make(map[string]interface{})
	}
	
	// Get CRD instance values with fallback
	instanceSpec, err := h.getSpecWithFallback(ctx, instanceName, detectedNamespace)
	if err != nil || instanceSpec == nil {
		instanceSpec = make(map[string]interface{})
	}
	
	// Ensure services map exists to avoid nil index errors in template
	if instanceSpec["services"] == nil {
		instanceSpec["services"] = make(map[string]interface{})
	}
	
	// Get service values for current values display
	serviceValues := h.getServiceValuesMap(ctx, services, detectedNamespace, instanceName)
	
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
		"CurrentInstance":    instanceName,     // Current instance name for template
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
	
	// Get namespace and instance name
	detectedNamespace, instanceName := h.getNamespaceAndInstance(r)
	
	ctx, cancel := context.WithTimeout(r.Context(), DefaultRequestTimeout)
	defer cancel()
	
	// Get CRD schema with fallback
	specSchema, err := h.getCRDSchemaWithFallback(ctx)
	if err != nil {
		h.logger.V(1).Info("failed to get CRD schema, using empty schema", "error", err)
		specSchema = make(map[string]interface{})
	}
	
	// Get instance values with fallback
	instanceSpec, err := h.getSpecWithFallback(ctx, instanceName, detectedNamespace)
	if err != nil || instanceSpec == nil {
		instanceSpec = make(map[string]interface{})
	}
	
	// Merge schema with instance values
	mergedSchema := mergeSchemaWithInstance(specSchema, instanceSpec)
	
	// Get service names for the template
	services := h.getServiceNames()
	
	data := map[string]interface{}{
		"Schema":          mergedSchema,
		"Services":        services,
		"CurrentInstance": instanceName,
		// AppName and AppVersion will be added by renderTemplate
	}
	
	if err := h.renderTemplate(w, "parameters", data); err != nil {
		h.logger.Error(err, "failed to render template")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "template_execution_failed", "Failed to execute template", nil)
	}
}

func (h *Handler) LogsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := h.renderTemplate(w, "logs", nil); err != nil {
		h.logger.Error(err, "failed to render template")
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "template_execution_failed", "Failed to execute template", nil)
	}
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
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	
	// Set cache headers - no cache for JS files to prevent stale code
	if ext == ".js" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	} else {
		// Cache other static files for 1 hour
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	
	// Copy file content to response
	io.Copy(w, file)
}
