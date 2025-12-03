package reconciler

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"github.com/garunski/conductor-framework/pkg/framework/events"
)

func (r *reconcilerImpl) ReconcileKey(ctx context.Context, key string) error {
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

func (r *reconcilerImpl) DeployAll(ctx context.Context) error {
	r.logger.Info("Deploying all manifests")
	r.reconcileAll(ctx)
	return nil
}

func (r *reconcilerImpl) DeleteAll(ctx context.Context) error {
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

func (r *reconcilerImpl) UpdateAll(ctx context.Context) error {
	r.logger.Info("Updating all manifests")
	r.reconcileAll(ctx)
	return nil
}


