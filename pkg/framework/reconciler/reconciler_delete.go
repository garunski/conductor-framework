package reconciler

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"github.com/garunski/conductor-framework/pkg/framework/events"
)

func (r *reconcilerImpl) deleteObject(ctx context.Context, obj *unstructured.Unstructured, resourceKey string) error {
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

