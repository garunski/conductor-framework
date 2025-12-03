package reconciler

import (
	"context"
	"testing"
)

func TestReconciler_ManagedKeys(t *testing.T) {
	rec := setupTestReconcilerForTests(t)
	ctx := context.Background()

	// Test isManaged
	if rec.isManaged("test/key") {
		t.Error("isManaged() should return false for unmanaged key")
	}

	// Test setManaged
	rec.setManaged("test/key")
	if !rec.isManaged("test/key") {
		t.Error("isManaged() should return true after setManaged")
	}

	// Test removeManaged
	rec.removeManaged("test/key")
	if rec.isManaged("test/key") {
		t.Error("isManaged() should return false after removeManaged")
	}

	// Test getAllManagedKeys
	rec.setManaged("key1")
	rec.setManaged("key2")
	keys := rec.getAllManagedKeys(ctx)
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
	rec.setAllManagedKeys(ctx, newKeys)
	keys = rec.getAllManagedKeys(ctx)
	if len(keys) != 2 {
		t.Errorf("setAllManagedKeys() did not replace keys, got %d keys", len(keys))
	}
	if !keys["key3"] || !keys["key4"] {
		t.Error("setAllManagedKeys() did not set correct keys")
	}

	// Test clearManagedKeys
	rec.clearManagedKeys(ctx)
	keys = rec.getAllManagedKeys(ctx)
	if len(keys) != 0 {
		t.Errorf("clearManagedKeys() did not clear keys, got %d keys", len(keys))
	}
}
