package reconciler

import (
	"context"
	"testing"
)

func TestReconciler_ManagedKeys(t *testing.T) {
	rec := setupTestReconcilerForTests(t)
	impl := getReconcilerImpl(t, rec)
	ctx := context.Background()

	// Test isManaged
	if impl.isManaged("test/key") {
		t.Error("isManaged() should return false for unmanaged key")
	}

	// Test setManaged
	impl.setManaged("test/key")
	if !impl.isManaged("test/key") {
		t.Error("isManaged() should return true after setManaged")
	}

	// Test removeManaged
	impl.removeManaged("test/key")
	if impl.isManaged("test/key") {
		t.Error("isManaged() should return false after removeManaged")
	}

	// Test getAllManagedKeys
	impl.setManaged("key1")
	impl.setManaged("key2")
	keys := impl.getAllManagedKeys(ctx)
	if len(keys) != 2 {
		t.Errorf("getAllManagedKeys() returned %d keys, want 2", len(keys))
	}
	if !keys["key1"] || !keys["key2"] {
		t.Error("getAllManagedKeys() missing expected keys")
	}

	// Test setAllManagedKeys
	newKeys := map[string]bool{
		"key3": true,
		"key4": true,
	}
	impl.setAllManagedKeys(ctx, newKeys)
	keys = impl.getAllManagedKeys(ctx)
	if len(keys) != 2 {
		t.Errorf("setAllManagedKeys() did not replace keys, got %d keys", len(keys))
	}
	if !keys["key3"] || !keys["key4"] {
		t.Error("setAllManagedKeys() did not set correct keys")
	}

	// Test clearManagedKeys
	impl.clearManagedKeys(ctx)
	keys = impl.getAllManagedKeys(ctx)
	if len(keys) != 0 {
		t.Errorf("clearManagedKeys() did not clear keys, got %d keys", len(keys))
	}
}
