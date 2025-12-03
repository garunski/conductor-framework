package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
