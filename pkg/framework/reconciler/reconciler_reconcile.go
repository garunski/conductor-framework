package reconciler

import (
	"context"
	"sync"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/garunski/conductor-framework/pkg/framework/events"
)

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

