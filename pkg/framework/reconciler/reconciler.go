package reconciler

import (
	"fmt"
	"os"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/go-logr/logr"
	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"github.com/garunski/conductor-framework/pkg/framework/store"
)

type reconcilerImpl struct {
	clientset         kubernetes.Interface
	dynamicClient     dynamic.Interface
	store             store.ManifestStore
	logger            logr.Logger
	scheme            *runtime.Scheme
	ready             bool
	managedKeys       sync.Map
	eventStore        events.EventStorage
	discoveryClient   discovery.DiscoveryInterface
	gvkCache          map[string]schema.GroupVersionKind
	resourceNameCache map[string]string
	cacheMu           sync.RWMutex
	firstReconcileCh  chan struct{}
	firstReconcileMu  sync.Mutex
	appName           string
}

func (r *reconcilerImpl) GetClientset() kubernetes.Interface {
	return r.clientset
}

func (r *reconcilerImpl) SetReady(ready bool) {
	r.ready = ready
}

func (r *reconcilerImpl) IsReady() bool {
	return r.ready
}

type ReconciliationResult struct {
	AppliedCount int
	FailedCount  int
	DeletedCount int
	ManagedKeys  map[string]bool
}

func GetKubernetesConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {

		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = clientcmd.RecommendedHomeFile
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("%w: kubernetes get config: failed to get Kubernetes config: %w", apperrors.ErrKubernetes, err)
		}
	}
	return config, nil
}

// NewReconciler creates a new Reconciler instance
// If appName is empty, it defaults to "conductor"
func NewReconciler(clientset kubernetes.Interface, dynamicClient dynamic.Interface, store store.ManifestStore, logger logr.Logger, eventStore events.EventStorage, appName string) (Reconciler, error) {
	// Default appName to "conductor" if not provided
	if appName == "" {
		appName = "conductor"
	}

	kubeScheme := runtime.NewScheme()
	if err := corev1.AddToScheme(kubeScheme); err != nil {
		return nil, fmt.Errorf("%w: kubernetes add corev1 to scheme: failed to add corev1 to scheme: %w", apperrors.ErrKubernetes, err)
	}
	if err := appsv1.AddToScheme(kubeScheme); err != nil {
		return nil, fmt.Errorf("%w: kubernetes add appsv1 to scheme: failed to add appsv1 to scheme: %w", apperrors.ErrKubernetes, err)
	}
	if err := scheme.AddToScheme(kubeScheme); err != nil {
		return nil, fmt.Errorf("%w: kubernetes add standard scheme: failed to add standard scheme: %w", apperrors.ErrKubernetes, err)
	}

	defaultGVKMap := map[string]schema.GroupVersionKind{
		"Deployment":            {Group: "apps", Version: "v1", Kind: "Deployment"},
		"StatefulSet":           {Group: "apps", Version: "v1", Kind: "StatefulSet"},
		"Service":               {Group: "", Version: "v1", Kind: "Service"},
		"ConfigMap":             {Group: "", Version: "v1", Kind: "ConfigMap"},
		"Secret":                {Group: "", Version: "v1", Kind: "Secret"},
		"Namespace":             {Group: "", Version: "v1", Kind: "Namespace"},
		"PersistentVolumeClaim": {Group: "", Version: "v1", Kind: "PersistentVolumeClaim"},
	}

	rec := &reconcilerImpl{
		clientset:         clientset,
		dynamicClient:     dynamicClient,
		store:             store,
		logger:            logger,
		scheme:            kubeScheme,
		ready:             false,
		eventStore:        eventStore,
		discoveryClient:   clientset.Discovery(),
		firstReconcileCh:  make(chan struct{}, 1),
		gvkCache:          defaultGVKMap,
		resourceNameCache: make(map[string]string),
		appName:           appName,
	}

	return rec, nil
}
