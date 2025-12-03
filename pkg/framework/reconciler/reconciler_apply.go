package reconciler

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"

	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"github.com/garunski/conductor-framework/pkg/framework/events"
)

func (r *reconcilerImpl) applyObject(ctx context.Context, obj runtime.Object, resourceKey string) error {

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

