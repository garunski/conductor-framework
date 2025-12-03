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
	// Use the store which already has embedded manifests loaded at startup
	manifests := h.store.List()
	serviceInfos := extractServices(manifests)
	
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
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
	manifests := h.store.List()
	serviceInfos := extractServices(manifests)
	statuses := make([]ServiceStatus, 0, len(serviceInfos))

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	type result struct {
		status ServiceStatus
		index  int
	}
	resultChan := make(chan result, len(serviceInfos))

	for i, svc := range serviceInfos {
		go func(idx int, service serviceInfo) {
			status := checkServiceHealth(ctx, client, service)
			resultChan <- result{status: status, index: idx}
		}(i, svc)
	}

	collected := make(map[int]ServiceStatus)
	timeout := time.After(5 * time.Second)
	remaining := len(serviceInfos)

	for remaining > 0 {
		select {
		case res := <-resultChan:
			collected[res.index] = res.status
			remaining--
		case <-timeout:
			timeout = nil
			for remaining > 0 {
				select {
				case res := <-resultChan:
					collected[res.index] = res.status
					remaining--
				default:
					remaining = 0
				}
			}
		case <-ctx.Done():
			for remaining > 0 {
				select {
				case res := <-resultChan:
					collected[res.index] = res.status
					remaining--
				default:
					remaining = 0
				}
			}
		}
	}

	for i := 0; i < len(serviceInfos); i++ {
		if status, ok := collected[i]; ok {
			statuses = append(statuses, status)
		} else {
			statuses = append(statuses, ServiceStatus{
				Name:        serviceInfos[i].Name,
				Namespace:   serviceInfos[i].Namespace,
				Port:        serviceInfos[i].Port,
				Status:      "unknown",
				LastChecked: time.Now(),
			})
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

	key := fmt.Sprintf("%s/Service/%s", namespace, serviceName)
	serviceManifest, found := h.store.Get(key)
	var manifests map[string][]byte
	if !found {

		manifests = h.store.List()
		if manifests == nil {
			WriteError(w, h.logger, fmt.Errorf("%w: service %s/%s", apperrors.ErrNotFound, namespace, serviceName))
			return
		}
		serviceManifest, found = extractServiceManifest(manifests, namespace, serviceName)
		if !found {
			WriteError(w, h.logger, fmt.Errorf("%w: service %s/%s", apperrors.ErrNotFound, namespace, serviceName))
			return
		}
	} else {

		manifests = h.store.List()
	}

	selector, err := extractServiceSelector(serviceManifest)
	if err != nil {
		h.logger.Error(err, "failed to extract service selector")
		WriteError(w, h.logger, fmt.Errorf("%w: invalid service: %w", apperrors.ErrInvalid, err))
		return
	}

	deploymentManifest := findMatchingDeployment(manifests, namespace, selector)

	envVars := []EnvVar{}
	if deploymentManifest != nil {
		envVars = extractEnvVars(deploymentManifest)
	}

	response := ServiceDetailsResponse{
		Name: serviceName,
		Env:  envVars,
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, response)
}
