package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	// Check Kubernetes version
	versionInfo, err := clientset.Discovery().ServerVersion()
	if err != nil {
		requirements = append(requirements, ClusterRequirement{
			Name:        "Kubernetes Version",
			Description: "Kubernetes cluster version must be 1.24 or higher",
			Status:      "fail",
			Message:     fmt.Sprintf("Unable to check version: %v", err),
			Required:    true,
		})
	} else {
		major, _ := strconv.Atoi(strings.TrimPrefix(versionInfo.Major, "v"))
		minor, _ := strconv.Atoi(strings.Split(versionInfo.Minor, "+")[0])
		versionStr := fmt.Sprintf("%s.%s", versionInfo.Major, versionInfo.Minor)
		
		if major > 1 || (major == 1 && minor >= 24) {
			requirements = append(requirements, ClusterRequirement{
				Name:        "Kubernetes Version",
				Description: "Kubernetes cluster version must be 1.24 or higher",
				Status:      "pass",
				Message:     fmt.Sprintf("Version %s meets requirement", versionStr),
				Required:    true,
			})
		} else {
			requirements = append(requirements, ClusterRequirement{
				Name:        "Kubernetes Version",
				Description: "Kubernetes cluster version must be 1.24 or higher",
				Status:      "fail",
				Message:     fmt.Sprintf("Version %s is below minimum required (1.24)", versionStr),
				Required:    true,
			})
		}
	}

	// Check for available nodes
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		requirements = append(requirements, ClusterRequirement{
			Name:        "Available Nodes",
			Description: "At least one node must be available and ready",
			Status:      "fail",
			Message:     fmt.Sprintf("Unable to list nodes: %v", err),
			Required:    true,
		})
	} else {
		readyNodes := 0
		for _, node := range nodes.Items {
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
					readyNodes++
					break
				}
			}
		}
		
		if readyNodes > 0 {
			requirements = append(requirements, ClusterRequirement{
				Name:        "Available Nodes",
				Description: "At least one node must be available and ready",
				Status:      "pass",
				Message:     fmt.Sprintf("%d node(s) ready", readyNodes),
				Required:    true,
			})
		} else {
			requirements = append(requirements, ClusterRequirement{
				Name:        "Available Nodes",
				Description: "At least one node must be available and ready",
				Status:      "fail",
				Message:     "No ready nodes found",
				Required:    true,
			})
		}
	}

	// Check for storage class
	storageClasses, err := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		requirements = append(requirements, ClusterRequirement{
			Name:        "Storage Class",
			Description: "At least one storage class must be available",
			Status:      "warning",
			Message:     fmt.Sprintf("Unable to check storage classes: %v", err),
			Required:    false,
		})
	} else {
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
			requirements = append(requirements, ClusterRequirement{
				Name:        "Storage Class",
				Description: "At least one storage class must be available",
				Status:      "pass",
				Message:     message,
				Required:    false,
			})
		} else {
			requirements = append(requirements, ClusterRequirement{
				Name:        "Storage Class",
				Description: "At least one storage class must be available",
				Status:      "warning",
				Message:     "No storage classes found (PVCs may not work)",
				Required:    false,
			})
		}
	}

	// Check cluster resources (CPU and Memory)
	if nodes != nil && len(nodes.Items) > 0 {
		totalCPU := resource.NewQuantity(0, resource.DecimalSI)
		totalMemory := resource.NewQuantity(0, resource.BinarySI)
		allocatableCPU := resource.NewQuantity(0, resource.DecimalSI)
		allocatableMemory := resource.NewQuantity(0, resource.BinarySI)

		for _, node := range nodes.Items {
			totalCPU.Add(node.Status.Capacity[corev1.ResourceCPU])
			totalMemory.Add(node.Status.Capacity[corev1.ResourceMemory])
			allocatableCPU.Add(node.Status.Allocatable[corev1.ResourceCPU])
			allocatableMemory.Add(node.Status.Allocatable[corev1.ResourceMemory])
		}

		// Minimum: 2 CPU cores, 4GB RAM
		minCPU := resource.MustParse("2")
		minMemory := resource.MustParse("4Gi")

		cpuStatus := "pass"
		cpuMessage := fmt.Sprintf("Total: %s, Allocatable: %s", totalCPU.String(), allocatableCPU.String())
		if allocatableCPU.Cmp(minCPU) < 0 {
			cpuStatus = "warning"
			cpuMessage = fmt.Sprintf("Low CPU: %s available (recommended: %s)", allocatableCPU.String(), minCPU.String())
		}

		memStatus := "pass"
		memMessage := fmt.Sprintf("Total: %s, Allocatable: %s", totalMemory.String(), allocatableMemory.String())
		if allocatableMemory.Cmp(minMemory) < 0 {
			memStatus = "warning"
			memMessage = fmt.Sprintf("Low Memory: %s available (recommended: %s)", allocatableMemory.String(), minMemory.String())
		}

		requirements = append(requirements, ClusterRequirement{
			Name:        "CPU Resources",
			Description: "At least 2 CPU cores recommended",
			Status:      cpuStatus,
			Message:     cpuMessage,
			Required:    false,
		})

		requirements = append(requirements, ClusterRequirement{
			Name:        "Memory Resources",
			Description: "At least 4GB RAM recommended",
			Status:      memStatus,
			Message:     memMessage,
			Required:    false,
		})
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

