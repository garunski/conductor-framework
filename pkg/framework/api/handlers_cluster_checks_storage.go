package api

import (
	"context"
	"fmt"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/garunski/conductor-framework/pkg/framework/manifest"
)

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

