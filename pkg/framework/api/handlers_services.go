package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type serviceInfo struct {
	Name      string
	Namespace string
	Port      int
}

func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), DefaultRequestTimeout)
	defer cancel()
	
	// Use the store which already has embedded manifests loaded at startup
	manifests := h.store.List()
	serviceInfos := extractServices(ctx, manifests)
	
	services := make([]ServiceInfo, 0, len(serviceInfos))
	for _, svc := range serviceInfos {
		// Check if service is installed in Kubernetes
		installed := false
		if h.reconciler != nil {
			clientset := h.reconciler.GetClientset()
			if clientset != nil {
				_, err := clientset.CoreV1().Services(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
				installed = err == nil && !k8serrors.IsNotFound(err)
			}
		}
		
		services = append(services, ServiceInfo{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Port:      svc.Port,
			Installed: installed,
		})
	}

	response := ServiceListResponse{
		Services: services,
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, response)
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), DefaultRequestTimeout)
	defer cancel()
	
	manifests := h.store.List()
	serviceInfos := extractServices(ctx, manifests)
	if len(serviceInfos) == 0 {
		WriteJSONResponse(w, h.logger, http.StatusOK, StatusResponse{Services: []ServiceStatus{}})
		return
	}

	client := &http.Client{
		Timeout: DefaultHealthCheckTimeout,
	}

	// Start async health checks for all services
	type indexedResult struct {
		index  int
		status ServiceStatus
	}
	resultChan := make(chan indexedResult, len(serviceInfos))
	
	for i, svc := range serviceInfos {
		go func(idx int, service serviceInfo) {
			statusChan := checkServiceHealthAsync(ctx, client, service)
			select {
			case status := <-statusChan:
				resultChan <- indexedResult{index: idx, status: status}
			case <-ctx.Done():
				resultChan <- indexedResult{
					index: idx,
					status: ServiceStatus{
						Name:        service.Name,
						Namespace:   service.Namespace,
						Port:        service.Port,
						Status:      "unknown",
						LastChecked: time.Now(),
					},
				}
			}
		}(i, svc)
	}

	// Collect all results
	collected := make(map[int]ServiceStatus, len(serviceInfos))
	for i := 0; i < len(serviceInfos); i++ {
		result := <-resultChan
		collected[result.index] = result.status
	}

	// Build statuses in order
	statuses := make([]ServiceStatus, len(serviceInfos))
	for i := range serviceInfos {
		if status, ok := collected[i]; ok {
			statuses[i] = status
		} else {
			statuses[i] = ServiceStatus{
				Name:        serviceInfos[i].Name,
				Namespace:   serviceInfos[i].Namespace,
				Port:        serviceInfos[i].Port,
				Status:      "unknown",
				LastChecked: time.Now(),
			}
		}
	}

	response := StatusResponse{
		Services: statuses,
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, response)
}

func (h *Handler) ServiceDetails(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	serviceName := chi.URLParam(r, "name")

	if err := ValidateNamespace(namespace); err != nil {
		WriteError(w, h.logger, err)
		return
	}
	if err := ValidateResourceName(serviceName); err != nil {
		WriteError(w, h.logger, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), DefaultRequestTimeout)
	defer cancel()
	
	key := fmt.Sprintf("%s/Service/%s", namespace, serviceName)
	serviceManifest, found := h.store.Get(key)
	var manifests map[string][]byte
	if !found {
		manifests = h.store.List()
		if manifests == nil {
			WriteError(w, h.logger, fmt.Errorf("%w: service %s/%s", apperrors.ErrNotFound, namespace, serviceName))
			return
		}
		serviceManifest, found = extractServiceManifest(ctx, manifests, namespace, serviceName)
		if !found {
			WriteError(w, h.logger, fmt.Errorf("%w: service %s/%s", apperrors.ErrNotFound, namespace, serviceName))
			return
		}
	} else {
		manifests = h.store.List()
	}

	selector, err := extractServiceSelector(ctx, serviceManifest)
	if err != nil {
		h.logger.Error(err, "failed to extract service selector")
		WriteError(w, h.logger, apperrors.WrapInvalid(err, "invalid service"))
		return
	}

	deploymentManifest := findMatchingDeployment(ctx, manifests, namespace, selector)

	envVars := []EnvVar{}
	if deploymentManifest != nil {
		envVars = extractEnvVars(ctx, deploymentManifest)
	}

	response := ServiceDetailsResponse{
		Name: serviceName,
		Env:  envVars,
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, response)
}
