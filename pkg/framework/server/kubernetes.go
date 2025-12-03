package server

import (
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/garunski/conductor-framework/pkg/framework/reconciler"
)

// NewKubernetesClients creates and returns Kubernetes clientset and dynamic client.
// It handles Kubernetes configuration retrieval and client initialization.
func NewKubernetesClients(cfg *Config, logger logr.Logger) (kubernetes.Interface, dynamic.Interface, error) {
	logger.Info("Setting up Kubernetes client")
	kubeConfig, err := reconciler.GetKubernetesConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return clientset, dynamicClient, nil
}

