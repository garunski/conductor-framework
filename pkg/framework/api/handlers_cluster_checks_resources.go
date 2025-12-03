package api

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/garunski/conductor-framework/pkg/framework/manifest"
)

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

