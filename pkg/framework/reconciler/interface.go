package reconciler

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

// Reconciler defines the interface for reconciliation operations.
// This interface allows for better testability and reduced coupling.
type Reconciler interface {
	// GetClientset returns the Kubernetes clientset
	GetClientset() kubernetes.Interface

	// IsReady returns whether the reconciler is ready to handle requests
	IsReady() bool

	// SetReady sets the ready state of the reconciler
	SetReady(ready bool)

	// ReconcileKey reconciles a single manifest by key
	ReconcileKey(ctx context.Context, key string) error

	// DeployManifests deploys the provided manifests to the cluster
	DeployManifests(ctx context.Context, manifests map[string][]byte) error

	// UpdateManifests updates the provided manifests in the cluster
	UpdateManifests(ctx context.Context, manifests map[string][]byte) error

	// DeleteManifests deletes the provided manifests from the cluster
	DeleteManifests(ctx context.Context, manifests map[string][]byte) error

	// DeleteAll deletes all managed resources
	DeleteAll(ctx context.Context) error

	// WaitForFirstReconciliation waits for the first reconciliation to complete
	WaitForFirstReconciliation(ctx context.Context) error
}

// Ensure *reconcilerImpl implements Reconciler interface
var _ Reconciler = (*reconcilerImpl)(nil)

