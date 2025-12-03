package reconciler

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"github.com/garunski/conductor-framework/pkg/framework/events"
)

// parseYAML parses YAML data into a runtime.Object
func (r *reconcilerImpl) parseYAML(yamlData []byte, resourceKey string) (runtime.Object, error) {
	decoder := serializer.NewCodecFactory(r.scheme).UniversalDeserializer()
	obj, _, err := decoder.Decode(yamlData, nil, nil)
	if err != nil {
		events.StoreEventSafe(r.eventStore, r.logger, events.Error(resourceKey, "parse", "Failed to parse manifest YAML", err))
		return nil, fmt.Errorf("%w: failed to decode YAML for resource %s: %w", apperrors.ErrInvalidYAML, resourceKey, err)
	}
	return obj, nil
}

// parseKey parses a resource key string into an unstructured object
func (r *reconcilerImpl) parseKey(key string) (*unstructured.Unstructured, error) {
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

