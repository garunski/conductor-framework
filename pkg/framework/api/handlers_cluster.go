package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"

	"github.com/garunski/conductor-framework/pkg/framework/manifest"
)

func (h *Handler) ClusterRequirements(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	requirements := []ClusterRequirement{}

	if h.reconciler == nil {
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "reconciler_not_available", "Reconciler not available", nil)
		return
	}

	clientset := h.reconciler.GetClientset()
	if clientset == nil {
		WriteErrorResponse(w, h.logger, http.StatusInternalServerError, "clientset_not_available", "Kubernetes client not available", nil)
		return
	}

	// Load and process application-defined requirements only
	appRequirements, err := manifest.LoadApplicationRequirements(h.manifestFS, h.manifestRoot)
	if err != nil {
		h.logger.Error(err, "Failed to load application requirements")
		// No requirements to show if loading fails
		response := ClusterRequirementsResponse{
			Requirements: []ClusterRequirement{},
			Overall:      "pass",
		}
		WriteJSONResponse(w, h.logger, http.StatusOK, response)
		return
	}

	// Get cluster information needed for application requirements
	versionInfo, _ := clientset.Discovery().ServerVersion()
	nodes, _ := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	storageClasses, _ := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})

	// Process application-defined requirements
	for _, appReq := range appRequirements {
		req := h.processApplicationRequirement(ctx, clientset, appReq, nodes, versionInfo, storageClasses)
		if req != nil {
			requirements = append(requirements, *req)
		}
	}

	// Determine overall status
	overall := "pass"
	for _, req := range requirements {
		if req.Required && req.Status == "fail" {
			overall = "fail"
			break
		} else if req.Status == "warning" && overall == "pass" {
			overall = "warning"
		}
	}

	response := ClusterRequirementsResponse{
		Requirements: requirements,
		Overall:      overall,
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, response)
}

// processApplicationRequirement processes an application-defined requirement and returns a ClusterRequirement
func (h *Handler) processApplicationRequirement(ctx context.Context, clientset kubernetes.Interface, appReq manifest.ApplicationRequirement, nodes *corev1.NodeList, versionInfo *version.Info, storageClasses *storagev1.StorageClassList) *ClusterRequirement {
	switch appReq.CheckType {
	case "kubernetes-version":
		return h.checkKubernetesVersion(appReq, clientset.Discovery(), versionInfo)
	case "node-count":
		return h.checkNodeCount(appReq, nodes)
	case "storage-class":
		return h.checkStorageClass(appReq, clientset, ctx, storageClasses)
	case "cpu":
		return h.checkCPU(appReq, nodes)
	case "memory":
		return h.checkMemory(appReq, nodes)
	default:
		// Unknown check type - return as warning
		return &ClusterRequirement{
			Name:        appReq.Name,
			Description: appReq.Description,
			Status:      "warning",
			Message:     fmt.Sprintf("Unknown check type: %s", appReq.CheckType),
			Required:    appReq.Required,
		}
	}
}

func (h *Handler) checkKubernetesVersion(appReq manifest.ApplicationRequirement, discovery interface{ ServerVersion() (*version.Info, error) }, versionInfo *version.Info) *ClusterRequirement {
	if versionInfo == nil {
		var err error
		versionInfo, err = discovery.ServerVersion()
		if err != nil {
			return &ClusterRequirement{
				Name:        appReq.Name,
				Description: appReq.Description,
				Status:      "fail",
				Message:     fmt.Sprintf("Unable to check version: %v", err),
				Required:    appReq.Required,
			}
		}
	}

	// Get minimum version from config, default to 1.24
	minVersion := "1.24"
	if minVer, ok := appReq.CheckConfig["minimumVersion"].(string); ok {
		minVersion = minVer
	}

	// Parse version
	major, _ := strconv.Atoi(strings.TrimPrefix(versionInfo.Major, "v"))
	minor, _ := strconv.Atoi(strings.Split(versionInfo.Minor, "+")[0])
	versionStr := fmt.Sprintf("%s.%s", versionInfo.Major, versionInfo.Minor)

	// Parse minimum version
	minParts := strings.Split(minVersion, ".")
	minMajor, _ := strconv.Atoi(minParts[0])
	minMinor := 0
	if len(minParts) > 1 {
		minMinor, _ = strconv.Atoi(minParts[1])
	}

	status := "pass"
	message := fmt.Sprintf("Version %s meets requirement (>= %s)", versionStr, minVersion)
	if major < minMajor || (major == minMajor && minor < minMinor) {
		status = "fail"
		message = fmt.Sprintf("Version %s is below minimum required (%s)", versionStr, minVersion)
	}

	return &ClusterRequirement{
		Name:        appReq.Name,
		Description: appReq.Description,
		Status:      status,
		Message:     message,
		Required:    appReq.Required,
	}
}

func (h *Handler) checkNodeCount(appReq manifest.ApplicationRequirement, nodes *corev1.NodeList) *ClusterRequirement {
	if nodes == nil {
		return &ClusterRequirement{
			Name:        appReq.Name,
			Description: appReq.Description,
			Status:      "fail",
			Message:     "Unable to check nodes: node list not available",
			Required:    appReq.Required,
		}
	}

	readyNodes := 0
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				readyNodes++
				break
			}
		}
	}

	minNodes := 1
	if min, ok := appReq.CheckConfig["minimum"].(int); ok {
		minNodes = min
	} else if minStr, ok := appReq.CheckConfig["minimum"].(string); ok {
		if parsed, err := strconv.Atoi(minStr); err == nil {
			minNodes = parsed
		}
	}

	status := "pass"
	message := fmt.Sprintf("%d node(s) ready (minimum: %d)", readyNodes, minNodes)
	if readyNodes < minNodes {
		status = "fail"
		message = fmt.Sprintf("Only %d node(s) ready, need at least %d", readyNodes, minNodes)
	}

	return &ClusterRequirement{
		Name:        appReq.Name,
		Description: appReq.Description,
		Status:      status,
		Message:     message,
		Required:    appReq.Required,
	}
}

func (h *Handler) checkStorageClass(appReq manifest.ApplicationRequirement, clientset kubernetes.Interface, ctx context.Context, storageClasses *storagev1.StorageClassList) *ClusterRequirement {
	if storageClasses == nil {
		var err error
		storageClasses, err = clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
		if err != nil {
			return &ClusterRequirement{
				Name:        appReq.Name,
				Description: appReq.Description,
				Status:      "warning",
				Message:     fmt.Sprintf("Unable to check storage classes: %v", err),
				Required:    appReq.Required,
			}
		}
	}

	requiredName, _ := appReq.CheckConfig["name"].(string)
	if requiredName != "" {
		// Check for specific storage class
		found := false
		for _, sc := range storageClasses.Items {
			if sc.Name == requiredName {
				found = true
				break
			}
		}
		if found {
			return &ClusterRequirement{
				Name:        appReq.Name,
				Description: appReq.Description,
				Status:      "pass",
				Message:     fmt.Sprintf("Storage class '%s' is available", requiredName),
				Required:    appReq.Required,
			}
		}
		return &ClusterRequirement{
			Name:        appReq.Name,
			Description: appReq.Description,
			Status:      "fail",
			Message:     fmt.Sprintf("Storage class '%s' not found", requiredName),
			Required:    appReq.Required,
		}
	}

	// Just check if any storage class exists
	if len(storageClasses.Items) > 0 {
		defaultSC := ""
		for _, sc := range storageClasses.Items {
			if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
				defaultSC = sc.Name
				break
			}
		}
		message := fmt.Sprintf("%d storage class(es) available", len(storageClasses.Items))
		if defaultSC != "" {
			message += fmt.Sprintf(" (default: %s)", defaultSC)
		}
		return &ClusterRequirement{
			Name:        appReq.Name,
			Description: appReq.Description,
			Status:      "pass",
			Message:     message,
			Required:    appReq.Required,
		}
	}

	return &ClusterRequirement{
		Name:        appReq.Name,
		Description: appReq.Description,
		Status:      "warning",
		Message:     "No storage classes found",
		Required:    appReq.Required,
	}
}

func (h *Handler) checkCPU(appReq manifest.ApplicationRequirement, nodes *corev1.NodeList) *ClusterRequirement {
	if nodes == nil || len(nodes.Items) == 0 {
		return &ClusterRequirement{
			Name:        appReq.Name,
			Description: appReq.Description,
			Status:      "fail",
			Message:     "Unable to check CPU: no nodes available",
			Required:    appReq.Required,
		}
	}

	allocatableCPU := resource.NewQuantity(0, resource.DecimalSI)
	for _, node := range nodes.Items {
		allocatableCPU.Add(node.Status.Allocatable[corev1.ResourceCPU])
	}

	minCPUStr := "2"
	if min, ok := appReq.CheckConfig["minimum"].(string); ok {
		minCPUStr = min
	}
	minCPU, err := resource.ParseQuantity(minCPUStr)
	if err != nil {
		minCPU = resource.MustParse("2")
	}

	status := "pass"
	message := fmt.Sprintf("Total: %s, Allocatable: %s", allocatableCPU.String(), allocatableCPU.String())
	if allocatableCPU.Cmp(minCPU) < 0 {
		status = "warning"
		message = fmt.Sprintf("Low CPU: %s available (required: %s)", allocatableCPU.String(), minCPU.String())
		if appReq.Required {
			status = "fail"
		}
	}

	return &ClusterRequirement{
		Name:        appReq.Name,
		Description: appReq.Description,
		Status:      status,
		Message:     message,
		Required:    appReq.Required,
	}
}

func (h *Handler) checkMemory(appReq manifest.ApplicationRequirement, nodes *corev1.NodeList) *ClusterRequirement {
	if nodes == nil || len(nodes.Items) == 0 {
		return &ClusterRequirement{
			Name:        appReq.Name,
			Description: appReq.Description,
			Status:      "fail",
			Message:     "Unable to check memory: no nodes available",
			Required:    appReq.Required,
		}
	}

	allocatableMemory := resource.NewQuantity(0, resource.BinarySI)
	for _, node := range nodes.Items {
		allocatableMemory.Add(node.Status.Allocatable[corev1.ResourceMemory])
	}

	minMemoryStr := "4Gi"
	if min, ok := appReq.CheckConfig["minimum"].(string); ok {
		minMemoryStr = min
	}
	minMemory, err := resource.ParseQuantity(minMemoryStr)
	if err != nil {
		minMemory = resource.MustParse("4Gi")
	}

	status := "pass"
	message := fmt.Sprintf("Total: %s, Allocatable: %s", allocatableMemory.String(), allocatableMemory.String())
	if allocatableMemory.Cmp(minMemory) < 0 {
		status = "warning"
		message = fmt.Sprintf("Low Memory: %s available (required: %s)", allocatableMemory.String(), minMemory.String())
		if appReq.Required {
			status = "fail"
		}
	}

	return &ClusterRequirement{
		Name:        appReq.Name,
		Description: appReq.Description,
		Status:      status,
		Message:     message,
		Required:    appReq.Required,
	}
}
