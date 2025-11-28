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
	"gopkg.in/yaml.v3"
)

type serviceInfo struct {
	Name      string
	Namespace string
	Port      int
}

// k8sObject represents a generic Kubernetes object
type k8sObject struct {
	Kind     string                 `yaml:"kind"`
	Metadata k8sMetadata            `yaml:"metadata"`
	Spec     map[string]interface{} `yaml:"spec"`
}

// k8sMetadata represents Kubernetes object metadata
type k8sMetadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// k8sServiceSpec represents a Service spec
type k8sServiceSpec struct {
	Ports    []k8sServicePort       `yaml:"ports"`
	Selector map[string]string      `yaml:"selector"`
	Other    map[string]interface{} `yaml:",inline"`
}

// k8sServicePort represents a Service port
type k8sServicePort struct {
	Port int `yaml:"port"`
}

// k8sDeploymentSpec represents a Deployment/StatefulSet spec
type k8sDeploymentSpec struct {
	Template k8sPodTemplateSpec     `yaml:"template"`
	Other    map[string]interface{} `yaml:",inline"`
}

// k8sPodTemplateSpec represents a pod template
type k8sPodTemplateSpec struct {
	Metadata k8sMetadata            `yaml:"metadata"`
	Spec     k8sPodSpec             `yaml:"spec"`
	Other    map[string]interface{} `yaml:",inline"`
}

// k8sPodSpec represents a pod spec
type k8sPodSpec struct {
	Containers []k8sContainer         `yaml:"containers"`
	Other      map[string]interface{} `yaml:",inline"`
}

// k8sContainer represents a container
type k8sContainer struct {
	Env   []k8sEnvVar            `yaml:"env"`
	Other map[string]interface{} `yaml:",inline"`
}

// k8sEnvVar represents an environment variable
type k8sEnvVar struct {
	Name      string                 `yaml:"name"`
	Value     string                 `yaml:"value"`
	ValueFrom map[string]interface{} `yaml:"valueFrom"`
}

func extractServices(manifests map[string][]byte) []serviceInfo {
	services := make([]serviceInfo, 0)

	for _, yamlData := range manifests {
		var obj k8sObject
		if err := yaml.Unmarshal(yamlData, &obj); err != nil {
			continue
		}

		if obj.Kind != "Service" {
			continue
		}

		if obj.Metadata.Name == "" {
			continue
		}

		namespace := "default"
		if obj.Metadata.Namespace != "" {
			namespace = obj.Metadata.Namespace
		}

		// Parse ports from spec
		if ports, ok := obj.Spec["ports"].([]interface{}); ok && len(ports) > 0 {
			if portObj, ok := ports[0].(map[string]interface{}); ok {
				port := 0
				if p, ok := portObj["port"].(int); ok {
					port = p
				} else if p, ok := portObj["port"].(float64); ok {
					port = int(p)
				}
				if port > 0 {
					services = append(services, serviceInfo{
						Name:      obj.Metadata.Name,
						Namespace: namespace,
						Port:      port,
					})
				}
			}
		}
	}

	return services
}

func checkServiceHealth(ctx context.Context, client *http.Client, svc serviceInfo) ServiceStatus {
	status := ServiceStatus{
		Name:        svc.Name,
		Namespace:   svc.Namespace,
		Port:        svc.Port,
		Status:      "unknown",
		LastChecked: time.Now(),
	}

	healthPaths := []string{"/health", "/healthz", "/readyz"}
	url := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", svc.Name, svc.Namespace, svc.Port)

	for _, path := range healthPaths {

		select {
		case <-ctx.Done():
			return status
		default:
		}

		fullURL := url + path
		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			status.Status = "healthy"
			status.Endpoint = path
			return status
		}
	}

	status.Status = "unhealthy"
	return status
}

func extractServiceManifest(manifests map[string][]byte, namespace, name string) ([]byte, bool) {
	for _, yamlData := range manifests {
		var obj k8sObject
		if err := yaml.Unmarshal(yamlData, &obj); err != nil {
			continue
		}

		if obj.Kind != "Service" {
			continue
		}

		if obj.Metadata.Name != name {
			continue
		}

		ns := "default"
		if obj.Metadata.Namespace != "" {
			ns = obj.Metadata.Namespace
		}

		if ns == namespace {
			return yamlData, true
		}
	}
	return nil, false
}

func extractServiceSelector(serviceManifest []byte) (map[string]string, error) {
	var serviceObj k8sObject
	if err := yaml.Unmarshal(serviceManifest, &serviceObj); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal service manifest: %w", apperrors.ErrInvalidYAML, err)
	}

	selectorRaw, ok := serviceObj.Spec["selector"]
	if !ok {
		return nil, fmt.Errorf("%w: service selector not found", apperrors.ErrInvalid)
	}

	selectorMap, ok := selectorRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: service selector is not a map", apperrors.ErrInvalid)
	}

	selector := make(map[string]string)
	for k, v := range selectorMap {
		if strVal, ok := v.(string); ok {
			selector[k] = strVal
		}
	}

	return selector, nil
}

func findMatchingDeployment(manifests map[string][]byte, namespace string, selector map[string]string) []byte {
	for _, yamlData := range manifests {
		var obj k8sObject
		if err := yaml.Unmarshal(yamlData, &obj); err != nil {
			continue
		}

		if obj.Kind != "Deployment" && obj.Kind != "StatefulSet" {
			continue
		}

		ns := "default"
		if obj.Metadata.Namespace != "" {
			ns = obj.Metadata.Namespace
		}

		if ns != namespace {
			continue
		}

		templateRaw, ok := obj.Spec["template"]
		if !ok {
			continue
		}

		templateMap, ok := templateRaw.(map[string]interface{})
		if !ok {
			continue
		}

		templateMetadataRaw, ok := templateMap["metadata"]
		if !ok {
			continue
		}

		templateMetadataMap, ok := templateMetadataRaw.(map[string]interface{})
		if !ok {
			continue
		}

		labelsRaw, ok := templateMetadataMap["labels"]
		if !ok {
			continue
		}

		labelsMap, ok := labelsRaw.(map[string]interface{})
		if !ok {
			continue
		}

		matches := true
		for key, value := range selector {
			if labelVal, ok := labelsMap[key].(string); !ok || labelVal != value {
				matches = false
				break
			}
		}

		if matches {
			return yamlData
		}
	}
	return nil
}

func extractEnvVars(deploymentManifest []byte) []EnvVar {
	var deployObj k8sObject
	if err := yaml.Unmarshal(deploymentManifest, &deployObj); err != nil {
		return nil
	}

	specRaw, ok := deployObj.Spec["template"]
	if !ok {
		return nil
	}

	templateMap, ok := specRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	templateSpecRaw, ok := templateMap["spec"]
	if !ok {
		return nil
	}

	templateSpecMap, ok := templateSpecRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	containersRaw, ok := templateSpecMap["containers"]
	if !ok {
		return nil
	}

	containers, ok := containersRaw.([]interface{})
	if !ok || len(containers) == 0 {
		return nil
	}

	container, ok := containers[0].(map[string]interface{})
	if !ok {
		return nil
	}

	envVarsRaw, ok := container["env"]
	if !ok {
		return nil
	}

	envVars, ok := envVarsRaw.([]interface{})
	if !ok {
		return nil
	}

	var result []EnvVar
	for _, envVarRaw := range envVars {
		env, ok := envVarRaw.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := env["name"].(string)
		if !ok {
			continue
		}

		envVarObj := EnvVar{Name: name}

		if value, ok := env["value"].(string); ok {
			envVarObj.Value = value
			envVarObj.Source = "direct"
		} else if valueFrom, ok := env["valueFrom"].(map[string]interface{}); ok {
			if secretRef, ok := valueFrom["secretKeyRef"].(map[string]interface{}); ok {
				secretName, _ := secretRef["name"].(string)
				key, _ := secretRef["key"].(string)
				envVarObj.Source = fmt.Sprintf("secret:%s/%s", secretName, key)
			} else if configMapRef, ok := valueFrom["configMapKeyRef"].(map[string]interface{}); ok {
				configMapName, _ := configMapRef["name"].(string)
				key, _ := configMapRef["key"].(string)
				envVarObj.Source = fmt.Sprintf("configmap:%s/%s", configMapName, key)
			} else {
				envVarObj.Source = "unknown"
			}
		}

		result = append(result, envVarObj)
	}

	return result
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
