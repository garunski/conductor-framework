package server

import (
	"context"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/garunski/conductor-framework/pkg/framework/crd"
)

// NewCRDClient creates and initializes a CRD parameter client.
// It also ensures a default DeploymentParameters instance exists.
func NewCRDClient(dynamicClient dynamic.Interface, cfg *Config, logger logr.Logger) (*crd.Client, error) {
	// Always initialize CRD parameter client
	parameterClient := crd.NewClient(dynamicClient, logger, cfg.CRDGroup, cfg.CRDVersion, cfg.CRDResource)
	logger.Info("CRD parameter client initialized", "group", cfg.CRDGroup, "version", cfg.CRDVersion, "resource", cfg.CRDResource)

	// Get or create default DeploymentParameters instance
	ctx := context.Background()
	defaultNamespace := "default"
	defaultParams, err := parameterClient.Get(ctx, crd.DefaultName, defaultNamespace)
	if err != nil {
		// Log error but continue - parameter client is still available, just can't get/create default instance
		logger.Error(err, "failed to get default DeploymentParameters, continuing without it")
	} else if defaultParams == nil {
		logger.Info("Creating default DeploymentParameters instance")
		defaultParams = &crd.DeploymentParameters{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crd.DefaultName,
				Namespace: defaultNamespace,
			},
			Spec: crd.DeploymentParametersSpec{
				"global": map[string]interface{}{
					"namespace":  "default",
					"namePrefix": "",
					"replicas":   int32(1),
				},
			},
		}
		if err := parameterClient.Create(ctx, defaultParams); err != nil {
			logger.Error(err, "failed to create default DeploymentParameters, continuing without it")
		} else {
			logger.Info("Created default DeploymentParameters instance")
		}
	}

	return parameterClient, nil
}

