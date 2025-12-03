package reconciler

import (
	"context"
)

// isManaged checks if a key is managed
func (r *reconcilerImpl) isManaged(key string) bool {
	_, ok := r.managedKeys.Load(key)
	return ok
}

// setManaged marks a key as managed
func (r *reconcilerImpl) setManaged(key string) {
	r.managedKeys.Store(key, true)
}

// removeManaged removes a key from managed keys
func (r *reconcilerImpl) removeManaged(key string) {
	r.managedKeys.Delete(key)
}

// getAllManagedKeys returns all managed keys as a map
func (r *reconcilerImpl) getAllManagedKeys(ctx context.Context) map[string]bool {
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
func (r *reconcilerImpl) setAllManagedKeys(ctx context.Context, keys map[string]bool) {
	r.managedKeys.Range(func(key, value interface{}) bool {
		r.managedKeys.Delete(key)
		return true
	})

	for key := range keys {
		r.managedKeys.Store(key, true)
	}
}

// clearManagedKeys removes all managed keys
func (r *reconcilerImpl) clearManagedKeys(ctx context.Context) {
	r.managedKeys.Range(func(key, value interface{}) bool {
		r.managedKeys.Delete(key)
		return true
	})
}

