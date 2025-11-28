package reconciler

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
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

type Reconciler struct {
	clientset         kubernetes.Interface
	dynamicClient     dynamic.Interface
	store             *store.ManifestStore
	logger            logr.Logger
	scheme            *runtime.Scheme
	ready             bool
	managedKeys       sync.Map
	eventStore        *events.Storage
	discoveryClient   discovery.DiscoveryInterface
	gvkCache          map[string]schema.GroupVersionKind
	resourceNameCache map[string]string
	cacheMu           sync.RWMutex
	firstReconcileCh  chan struct{}
	firstReconcileMu  sync.Mutex
	appName           string
}

func (r *Reconciler) GetClientset() kubernetes.Interface {
	return r.clientset
}

func (r *Reconciler) SetReady(ready bool) {
	r.ready = ready
}

func (r *Reconciler) IsReady() bool {
	return r.ready
}

type ReconciliationResult struct {
	AppliedCount int
	FailedCount  int
	DeletedCount int
	ManagedKeys  map[string]bool
}

func (r *Reconciler) applyObject(ctx context.Context, obj runtime.Object, resourceKey string) error {

	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		// Get the GVK from the typed object
		gvks, _, err := r.scheme.ObjectKinds(obj)
		if err != nil || len(gvks) == 0 {
			err := fmt.Errorf("%w: kubernetes convert to unstructured %s: failed to get object kinds: %w", apperrors.ErrKubernetes, resourceKey, err)
			events.StoreEventSafe(r.eventStore, r.logger, events.Error(resourceKey, "apply", "Failed to apply object", err))
			return err
		}
		
		// Use the first GVK found and create codec with proper GroupVersion
		gvk := gvks[0]
		codec := serializer.NewCodecFactory(r.scheme).LegacyCodec(gvk.GroupVersion())
		
		unstructuredObj = &unstructured.Unstructured{}
		
		data, err := runtime.Encode(codec, obj)
		if err != nil {
			err := fmt.Errorf("%w: kubernetes encode object %s: failed to encode object: %w", apperrors.ErrKubernetes, resourceKey, err)
			events.StoreEventSafe(r.eventStore, r.logger, events.Error(resourceKey, "apply", "Failed to apply object", err))
			return err
		}

		_, _, err = serializer.NewCodecFactory(r.scheme).UniversalDeserializer().Decode(data, nil, unstructuredObj)
		if err != nil {
			err := fmt.Errorf("%w: kubernetes convert to unstructured %s: failed to convert object to unstructured: %w", apperrors.ErrKubernetes, resourceKey, err)
			events.StoreEventSafe(r.eventStore, r.logger, events.Error(resourceKey, "apply", "Failed to apply object", err))
			return err
		}
	}

	gvk := unstructuredObj.GroupVersionKind()
	if gvk.Kind == "" {
		err := fmt.Errorf("%w: object missing kind for resource %s", apperrors.ErrInvalid, resourceKey)
		events.StoreEventSafe(r.eventStore, r.logger, events.Error(resourceKey, "apply", "Failed to apply object", err))
		return err
	}

	resource := r.resolveResourceName(gvk)

	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}

	var resourceInterface dynamic.ResourceInterface
	if unstructuredObj.GetNamespace() != "" {
		resourceInterface = r.dynamicClient.Resource(gvr).Namespace(unstructuredObj.GetNamespace())
	} else {
		resourceInterface = r.dynamicClient.Resource(gvr)
	}

	_, err := resourceInterface.Apply(ctx, unstructuredObj.GetName(), unstructuredObj, metav1.ApplyOptions{FieldManager: r.appName, Force: true})
	if err != nil {
		events.StoreEventSafe(r.eventStore, r.logger, events.Error(resourceKey, "apply", "Failed to apply manifest to cluster", err))
		return fmt.Errorf("%w: kubernetes apply %s: failed to apply resource: %w", apperrors.ErrKubernetes, resourceKey, err)
	}

	events.StoreEventSafe(r.eventStore, r.logger, events.Success(resourceKey, "apply", "Successfully applied manifest"))
	return nil
}

func (r *Reconciler) deleteObject(ctx context.Context, obj *unstructured.Unstructured, resourceKey string) error {
	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" {
		return fmt.Errorf("%w: object missing kind for resource %s", apperrors.ErrInvalid, resourceKey)
	}

	resource := r.resolveResourceName(gvk)

	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}

	var resourceInterface dynamic.ResourceInterface
	if obj.GetNamespace() != "" {
		resourceInterface = r.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		resourceInterface = r.dynamicClient.Resource(gvr)
	}

	err := resourceInterface.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		err = fmt.Errorf("%w: kubernetes delete %s: failed to delete resource: %w", apperrors.ErrKubernetes, resourceKey, err)
		events.StoreEventSafe(r.eventStore, r.logger, events.Error(resourceKey, "delete", "Failed to delete resource from cluster", err))
		return err
	}

	if k8serrors.IsNotFound(err) {
		r.logger.Info("Resource already deleted from cluster", "key", resourceKey)
		events.StoreEventSafe(r.eventStore, r.logger, events.Info(resourceKey, "delete", "Resource already deleted from cluster"))
	} else {
		r.logger.Info("Successfully deleted resource", "key", resourceKey)
		events.StoreEventSafe(r.eventStore, r.logger, events.Success(resourceKey, "delete", "Successfully deleted resource"))
	}
	return nil
}

func (r *Reconciler) reconcile(ctx context.Context, manifests map[string][]byte, previousKeys map[string]bool) (ReconciliationResult, error) {
	currentKeys := make(map[string]bool)
	appliedCount := 0
	failedCount := 0

	for key := range manifests {
		currentKeys[key] = true
	}

	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for key, yamlData := range manifests {
		wg.Add(1)
		go func(key string, yamlData []byte) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			obj, err := r.parseYAML(yamlData, key)
			if err != nil {
				r.logger.Error(err, "failed to parse manifest YAML", "key", key, "error", err.Error())
				mu.Lock()
				failedCount++
				mu.Unlock()
				return
			}

			if err := r.applyObject(ctx, obj, key); err != nil {
				r.logger.Error(err, "failed to apply manifest to cluster", "key", key, "error", err.Error())
				mu.Lock()
				failedCount++
				mu.Unlock()
			} else {
				mu.Lock()
				currentKeys[key] = true
				appliedCount++
				mu.Unlock()
			}
		}(key, yamlData)
	}

	wg.Wait()

	deletedCount := r.deleteOrphanedResources(ctx, previousKeys, currentKeys)

	return ReconciliationResult{
		AppliedCount: appliedCount,
		FailedCount:  failedCount,
		DeletedCount: deletedCount,
		ManagedKeys:  currentKeys,
	}, nil
}

func (r *Reconciler) deleteOrphanedResources(ctx context.Context, previousKeys, currentKeys map[string]bool) int {
	deletedCount := 0
	for key := range previousKeys {
		if !currentKeys[key] {

			obj, err := r.parseKey(key)
			if err != nil {
				r.logger.Error(err, "failed to parse key for deletion", "key", key, "error", err.Error())
				events.StoreEventSafe(r.eventStore, r.logger, events.Error(key, "delete", "Failed to parse key for deletion", err))
				continue
			}

			if err := r.deleteObject(ctx, obj, key); err != nil {
				if !k8serrors.IsNotFound(err) {
					r.logger.Error(err, "failed to delete resource from cluster", "key", key, "error", err.Error())
				} else {
					deletedCount++
				}
			} else {
				deletedCount++
			}
		}
	}
	return deletedCount
}

// parseYAML parses YAML data into a runtime.Object
func (r *Reconciler) parseYAML(yamlData []byte, resourceKey string) (runtime.Object, error) {
	decoder := serializer.NewCodecFactory(r.scheme).UniversalDeserializer()
	obj, _, err := decoder.Decode(yamlData, nil, nil)
	if err != nil {
		events.StoreEventSafe(r.eventStore, r.logger, events.Error(resourceKey, "parse", "Failed to parse manifest YAML", err))
		return nil, fmt.Errorf("%w: failed to decode YAML for resource %s: %w", apperrors.ErrInvalidYAML, resourceKey, err)
	}
	return obj, nil
}

// parseKey parses a resource key string into an unstructured object
func (r *Reconciler) parseKey(key string) (*unstructured.Unstructured, error) {
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: invalid key format: %s (expected namespace/kind/name)", apperrors.ErrInvalid, key)
	}

	namespace := parts[0]
	kind := parts[1]
	name := parts[2]

	gvk, err := r.resolveGVK(kind)
	if err != nil {
		r.logger.V(1).Info("failed to resolve GVK, using generic", "kind", kind, "error", err)
		gvk = schema.GroupVersionKind{Kind: kind}
	}

	obj := r.getObjectForGVK(gvk, namespace, name)
	return obj, nil
}

// resolveGVK resolves a kind string to a GroupVersionKind
func (r *Reconciler) resolveGVK(kind string) (schema.GroupVersionKind, error) {
	r.cacheMu.RLock()
	if gvk, found := r.gvkCache[kind]; found {
		r.cacheMu.RUnlock()
		return gvk, nil
	}
	r.cacheMu.RUnlock()

	if r.discoveryClient != nil {
		apiResourceLists, err := r.discoveryClient.ServerPreferredResources()
		if err != nil {
			r.logger.V(1).Info("failed to discover resources, will use unstructured", "error", err)
		} else {
			for _, apiResourceList := range apiResourceLists {
				gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
				if err != nil {
					continue
				}
				for _, apiResource := range apiResourceList.APIResources {
					if apiResource.Kind == kind {
						gvk := schema.GroupVersionKind{
							Group:   gv.Group,
							Version: gv.Version,
							Kind:    kind,
						}

						r.cacheMu.Lock()
						r.gvkCache[kind] = gvk
						r.cacheMu.Unlock()
						return gvk, nil
					}
				}
			}
		}
	}

	gvk := schema.GroupVersionKind{Kind: kind}
	r.cacheMu.Lock()
	r.gvkCache[kind] = gvk
	r.cacheMu.Unlock()
	return gvk, nil
}

// getObjectForGVK creates an unstructured object for the given GVK, namespace, and name
func (r *Reconciler) getObjectForGVK(gvk schema.GroupVersionKind, namespace, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(name)
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	return obj
}

// resolveResourceName resolves a GVK to its resource name (plural form)
func (r *Reconciler) resolveResourceName(gvk schema.GroupVersionKind) string {
	cacheKey := gvk.String()

	r.cacheMu.RLock()
	if resource, found := r.resourceNameCache[cacheKey]; found {
		r.cacheMu.RUnlock()
		return resource
	}
	r.cacheMu.RUnlock()

	resource := strings.ToLower(gvk.Kind) + "s"
	if r.discoveryClient != nil {
		apiResourceLists, err := r.discoveryClient.ServerPreferredResources()
		if err == nil {
			for _, apiResourceList := range apiResourceLists {
				gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
				if err != nil {
					continue
				}
				if gv.Group != gvk.Group || gv.Version != gvk.Version {
					continue
				}
				for _, apiResource := range apiResourceList.APIResources {
					if apiResource.Kind == gvk.Kind {
						resource = apiResource.Name
						break
					}
				}
			}
		}
	}

	r.cacheMu.Lock()
	r.resourceNameCache[cacheKey] = resource
	r.cacheMu.Unlock()

	return resource
}

// initCommonGVKs initializes the common GVK cache
func (r *Reconciler) initCommonGVKs() {
	r.gvkCache = map[string]schema.GroupVersionKind{
		"Deployment":            {Group: "apps", Version: "v1", Kind: "Deployment"},
		"StatefulSet":           {Group: "apps", Version: "v1", Kind: "StatefulSet"},
		"Service":               {Group: "", Version: "v1", Kind: "Service"},
		"ConfigMap":             {Group: "", Version: "v1", Kind: "ConfigMap"},
		"Secret":                {Group: "", Version: "v1", Kind: "Secret"},
		"Namespace":             {Group: "", Version: "v1", Kind: "Namespace"},
		"PersistentVolumeClaim": {Group: "", Version: "v1", Kind: "PersistentVolumeClaim"},
	}
	r.resourceNameCache = make(map[string]string)
}

// isManaged checks if a key is managed
func (r *Reconciler) isManaged(key string) bool {
	_, ok := r.managedKeys.Load(key)
	return ok
}

// setManaged marks a key as managed
func (r *Reconciler) setManaged(key string) {
	r.managedKeys.Store(key, true)
}

// removeManaged removes a key from managed keys
func (r *Reconciler) removeManaged(key string) {
	r.managedKeys.Delete(key)
}

// getAllManagedKeys returns all managed keys as a map
func (r *Reconciler) getAllManagedKeys(ctx context.Context) map[string]bool {
	result := make(map[string]bool)
	r.managedKeys.Range(func(key, value interface{}) bool {
		if strKey, ok := key.(string); ok {
			result[strKey] = true
		}
		return true
	})
	return result
}

// setAllManagedKeys replaces all managed keys with the given set
func (r *Reconciler) setAllManagedKeys(ctx context.Context, keys map[string]bool) {
	r.managedKeys.Range(func(key, value interface{}) bool {
		r.managedKeys.Delete(key)
		return true
	})

	for key := range keys {
		r.managedKeys.Store(key, true)
	}
}

// clearManagedKeys removes all managed keys
func (r *Reconciler) clearManagedKeys(ctx context.Context) {
	r.managedKeys.Range(func(key, value interface{}) bool {
		r.managedKeys.Delete(key)
		return true
	})
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
func NewReconciler(clientset kubernetes.Interface, dynamicClient dynamic.Interface, store *store.ManifestStore, logger logr.Logger, eventStore *events.Storage, appName string) (*Reconciler, error) {
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

	rec := &Reconciler{
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

func (r *Reconciler) reconcileAll(ctx context.Context) {
	manifests := r.store.List()

	events.StoreEventSafe(r.eventStore, r.logger, events.Info("", "reconcile", "Reconciliation started"))

	previousKeys := r.getAllManagedKeys(ctx)

	result, err := r.reconcile(ctx, manifests, previousKeys)
	if err != nil {
		r.logger.Error(err, "reconciliation failed")
		events.StoreEventSafe(r.eventStore, r.logger, events.Error("", "reconcile", "Reconciliation failed", err))
		return
	}

	r.setAllManagedKeys(ctx, result.ManagedKeys)

	// Signal first reconciliation completion
	r.firstReconcileMu.Lock()
	if !r.ready {
		r.ready = true
		select {
		case r.firstReconcileCh <- struct{}{}:
		default:
		}
	}
	r.firstReconcileMu.Unlock()

	event := events.Info("", "reconcile", "Reconciliation complete")
	event.Details["total"] = len(manifests)
	event.Details["applied"] = result.AppliedCount
	event.Details["failed"] = result.FailedCount
	event.Details["deleted"] = result.DeletedCount
	event.Details["managed"] = len(result.ManagedKeys)
	events.StoreEventSafe(r.eventStore, r.logger, event)

	r.logger.Info("Reconciliation complete",
		"total", len(manifests),
		"applied", result.AppliedCount,
		"failed", result.FailedCount,
		"deleted", result.DeletedCount,
		"managed", len(result.ManagedKeys))
}

func (r *Reconciler) ReconcileKey(ctx context.Context, key string) error {
	yamlData, ok := r.store.Get(key)
	if !ok {

		wasManaged := r.isManaged(key)

		if wasManaged {

			obj, err := r.parseKey(key)
			if err != nil {
				err = fmt.Errorf("%w: reconciliation failed for resource %s: failed to parse key for deletion: %w", apperrors.ErrReconciliation, key, err)
				events.StoreEventSafe(r.eventStore, r.logger, events.Error(key, "delete", "Failed to parse key for deletion", err))
				return err
			}

			if err := r.deleteObject(ctx, obj, key); err != nil && !k8serrors.IsNotFound(err) {
				err = fmt.Errorf("%w: reconciliation failed for resource %s: failed to delete resource: %w", apperrors.ErrReconciliation, key, err)
				events.StoreEventSafe(r.eventStore, r.logger, events.Error(key, "delete", "Failed to delete resource", err))
				return err
			}

			r.removeManaged(key)

			r.logger.Info("Deleted resource", "key", key)
			events.StoreEventSafe(r.eventStore, r.logger, events.Success(key, "delete", "Deleted resource"))
		}
		return nil
	}

	obj, err := r.parseYAML(yamlData, key)
	if err != nil {
		return fmt.Errorf("%w: reconciliation failed for manifest %s: failed to parse YAML: %w", apperrors.ErrReconciliation, key, err)
	}

	if err := r.applyObject(ctx, obj, key); err != nil {
		return err
	}

	r.setManaged(key)

	return nil
}

func (r *Reconciler) DeployAll(ctx context.Context) error {
	r.logger.Info("Deploying all manifests")
	r.reconcileAll(ctx)
	return nil
}

func (r *Reconciler) DeleteAll(ctx context.Context) error {
	r.logger.Info("Deleting all managed resources")

	manifests := r.store.List()
	keys := make([]string, 0, len(manifests))
	for key := range manifests {
		keys = append(keys, key)
	}

	managedKeys := r.getAllManagedKeys(ctx)
	for key := range managedKeys {
		if _, exists := manifests[key]; !exists {
			keys = append(keys, key)
		}
	}

	deletedCount := 0
	failedCount := 0
	for _, key := range keys {
		obj, err := r.parseKey(key)
		if err != nil {
			r.logger.Error(err, "failed to parse key for deletion", "key", key)
			events.StoreEventSafe(r.eventStore, r.logger, events.Error(key, "delete", "Failed to parse key for deletion", err))
			failedCount++
			continue
		}

		if err := r.deleteObject(ctx, obj, key); err != nil {
			if !k8serrors.IsNotFound(err) {
				r.logger.Error(err, "failed to delete resource", "key", key)
				failedCount++
			} else {
				deletedCount++
			}
		} else {
			deletedCount++
		}
	}

	r.clearManagedKeys(ctx)

	r.logger.Info("Deleted all managed resources", "count", deletedCount, "failed", failedCount, "total", len(keys))
	return nil
}

func (r *Reconciler) UpdateAll(ctx context.Context) error {
	r.logger.Info("Updating all manifests")
	r.reconcileAll(ctx)
	return nil
}

// DeployManifests deploys the provided manifests
func (r *Reconciler) DeployManifests(ctx context.Context, manifests map[string][]byte) error {
	r.logger.Info("Deploying selected manifests", "count", len(manifests))

	events.StoreEventSafe(r.eventStore, r.logger, events.Info("", "reconcile", "Reconciliation started"))

	previousKeys := r.getAllManagedKeys(ctx)

	result, err := r.reconcile(ctx, manifests, previousKeys)
	if err != nil {
		r.logger.Error(err, "reconciliation failed")
		events.StoreEventSafe(r.eventStore, r.logger, events.Error("", "reconcile", "Reconciliation failed", err))
		return err
	}

	// Update managed keys - add new ones but don't remove ones not in the filtered set
	for key := range result.ManagedKeys {
		r.setManaged(key)
	}

	event := events.Info("", "reconcile", "Reconciliation complete")
	event.Details["total"] = len(manifests)
	event.Details["applied"] = result.AppliedCount
	event.Details["failed"] = result.FailedCount
	event.Details["deleted"] = result.DeletedCount
	event.Details["managed"] = len(result.ManagedKeys)
	events.StoreEventSafe(r.eventStore, r.logger, event)

	r.logger.Info("Reconciliation complete",
		"total", len(manifests),
		"applied", result.AppliedCount,
		"failed", result.FailedCount,
		"deleted", result.DeletedCount,
		"managed", len(result.ManagedKeys))

	return nil
}

// UpdateManifests updates the provided manifests
func (r *Reconciler) UpdateManifests(ctx context.Context, manifests map[string][]byte) error {
	r.logger.Info("Updating selected manifests", "count", len(manifests))
	return r.DeployManifests(ctx, manifests)
}

// DeleteManifests deletes only the resources in the provided manifest map
func (r *Reconciler) DeleteManifests(ctx context.Context, manifests map[string][]byte) error {
	r.logger.Info("Deleting selected manifests", "count", len(manifests))

	keys := make([]string, 0, len(manifests))
	for key := range manifests {
		keys = append(keys, key)
	}

	deletedCount := 0
	failedCount := 0
	for _, key := range keys {
		obj, err := r.parseKey(key)
		if err != nil {
			r.logger.Error(err, "failed to parse key for deletion", "key", key)
			events.StoreEventSafe(r.eventStore, r.logger, events.Error(key, "delete", "Failed to parse key for deletion", err))
			failedCount++
			continue
		}

		if err := r.deleteObject(ctx, obj, key); err != nil {
			if !k8serrors.IsNotFound(err) {
				r.logger.Error(err, "failed to delete resource", "key", key)
				failedCount++
			} else {
				deletedCount++
			}
		} else {
			deletedCount++
		}

		// Remove from managed keys
		r.removeManaged(key)
	}

	r.logger.Info("Deleted selected manifests", "count", deletedCount, "failed", failedCount, "total", len(keys))
	return nil
}

func (r *Reconciler) StartPeriodicReconciliation(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Do initial reconciliation immediately
	r.reconcileAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.reconcileAll(ctx)
		}
	}
}

// WaitForFirstReconciliation waits for the first reconciliation to complete.
// This is useful for testing to ensure the reconciler is ready.
func (r *Reconciler) WaitForFirstReconciliation(ctx context.Context) error {
	select {
	case <-r.firstReconcileCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

