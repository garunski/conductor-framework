package reconciler

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/garunski/conductor-framework/pkg/framework/events"
)

// DeployManifests deploys the provided manifests
func (r *reconcilerImpl) DeployManifests(ctx context.Context, manifests map[string][]byte) error {
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
func (r *reconcilerImpl) UpdateManifests(ctx context.Context, manifests map[string][]byte) error {
	r.logger.Info("Updating selected manifests", "count", len(manifests))
	return r.DeployManifests(ctx, manifests)
}

// DeleteManifests deletes only the resources in the provided manifest map
func (r *reconcilerImpl) DeleteManifests(ctx context.Context, manifests map[string][]byte) error {
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

