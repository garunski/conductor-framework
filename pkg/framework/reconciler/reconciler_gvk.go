package reconciler

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// resolveGVK resolves a kind string to a GroupVersionKind
func (r *reconcilerImpl) resolveGVK(kind string) (schema.GroupVersionKind, error) {
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
func (r *reconcilerImpl) getObjectForGVK(gvk schema.GroupVersionKind, namespace, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(name)
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	return obj
}

// resolveResourceName resolves a GVK to its resource name (plural form)
func (r *reconcilerImpl) resolveResourceName(gvk schema.GroupVersionKind) string {
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
func (r *reconcilerImpl) initCommonGVKs() {
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

