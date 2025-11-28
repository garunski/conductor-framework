package store

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/garunski/conductor-framework/pkg/framework/database"
	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"github.com/garunski/conductor-framework/pkg/framework/index"
)

func TestManifestStore_Create(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	key := "test-key"
	value := []byte("test-value")

	if err := store.Create(key, value); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if stored, ok := store.Get(key); !ok {
		t.Error("manifest not found in index after Create")
	} else if string(stored) != string(value) {
		t.Errorf("stored value mismatch: got %s, want %s", string(stored), string(value))
	}

	if stored, err := db.Get(key); err != nil {
		t.Errorf("manifest not found in DB after Create: %v", err)
	} else if string(stored) != string(value) {
		t.Errorf("DB value mismatch: got %s, want %s", string(stored), string(value))
	}
}

func TestManifestStore_Update(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	key := "test-key"
	initialValue := []byte("initial-value")
	updatedValue := []byte("updated-value")

	if err := store.Create(key, initialValue); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if err := store.Update(key, updatedValue); err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	if stored, ok := store.Get(key); !ok {
		t.Error("manifest not found in index after Update")
	} else if string(stored) != string(updatedValue) {
		t.Errorf("index value mismatch: got %s, want %s", string(stored), string(updatedValue))
	}

	if stored, err := db.Get(key); err != nil {
		t.Errorf("manifest not found in DB after Update: %v", err)
	} else if string(stored) != string(updatedValue) {
		t.Errorf("DB value mismatch: got %s, want %s", string(stored), string(updatedValue))
	}
}

func TestManifestStore_Update_NotFound(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	key := "non-existent-key"
	value := []byte("value")

	err = store.Update(key, value)
	if err == nil {
		t.Error("Update() should fail for non-existent key")
	}
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManifestStore_Delete(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	key := "test-key"
	value := []byte("test-value")

	if err := store.Create(key, value); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if _, ok := store.Get(key); ok {
		t.Error("manifest still found in index after Delete")
	}

	if _, err := db.Get(key); err == nil {
		t.Error("manifest still found in DB after Delete")
	}
}

func TestManifestStore_Delete_NotFound(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	key := "non-existent-key"

	err = store.Delete(key)
	if err == nil {
		t.Error("Delete() should fail for non-existent key")
	}
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManifestStore_List(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	manifests := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	for key, value := range manifests {
		if err := store.Create(key, value); err != nil {
			t.Fatalf("Create() failed for %s: %v", key, err)
		}
	}

	list := store.List()
	if len(list) != len(manifests) {
		t.Errorf("List() returned %d items, want %d", len(list), len(manifests))
	}

	for key, expectedValue := range manifests {
		if actualValue, ok := list[key]; !ok {
			t.Errorf("manifest %s not found in List()", key)
		} else if string(actualValue) != string(expectedValue) {
			t.Errorf("manifest %s value mismatch: got %s, want %s", key, string(actualValue), string(expectedValue))
		}
	}
}

func TestManifestStore_Atomicity(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	key := "test-key"
	value := []byte("test-value")

	if err := store.Create(key, value); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	indexValue, indexOk := store.Get(key)
	dbValue, dbErr := db.Get(key)

	if !indexOk {
		t.Error("manifest not found in index")
	}
	if dbErr != nil {
		t.Errorf("manifest not found in DB: %v", dbErr)
	}
	if string(indexValue) != string(dbValue) {
		t.Errorf("index and DB values are inconsistent: index=%s, db=%s", string(indexValue), string(dbValue))
	}
}

func TestManifestStore_EdgeCases(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	err = store.Create("", []byte("value"))
	if err == nil {
		t.Error("Create() should handle empty key appropriately")
	}

	err = store.Create("key", []byte(""))
	if err != nil {
		t.Errorf("Create() with empty value should succeed, got error: %v", err)
	}

	largeValue := make([]byte, 10000)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}
	err = store.Create("large-key", largeValue)
	if err != nil {
		t.Errorf("Create() with large value should succeed, got error: %v", err)
	}

	specialKey := "key/with/special-chars-123"
	err = store.Create(specialKey, []byte("value"))
	if err != nil {
		t.Errorf("Create() with special characters in key should succeed, got error: %v", err)
	}

	emptyStore := NewManifestStore(db, index.NewIndex(), logger)
	list := emptyStore.List()
	if list == nil {
		t.Error("List() should return empty map, not nil")
	}
	if len(list) != 0 {
		t.Errorf("List() with empty store should return empty map, got %d items", len(list))
	}

	_, ok := store.Get("non-existent")
	if ok {
		t.Error("Get() with non-existent key should return false")
	}
}

func TestManifestStore_ConcurrentAccess(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func(n int) {
			key := fmt.Sprintf("key-%d", n)
			value := []byte(fmt.Sprintf("value-%d", n))
			err := store.Create(key, value)
			if err != nil {
				t.Errorf("Create() error: %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	list := store.List()
	if len(list) != 50 {
		t.Errorf("Expected 50 manifests, got %d", len(list))
	}

	done = make(chan bool, 100)
	for i := 0; i < 50; i++ {
		go func(n int) {
			key := fmt.Sprintf("key-%d", n)
			_, _ = store.Get(key)
			done <- true
		}(i)
		go func(n int) {
			key := fmt.Sprintf("new-key-%d", n)
			value := []byte(fmt.Sprintf("new-value-%d", n))
			_ = store.Create(key, value)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	list = store.List()
	if len(list) < 50 {
		t.Errorf("Expected at least 50 manifests after concurrent operations, got %d", len(list))
	}
}

func TestManifestStore_ConcurrentOperations(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		_ = store.Create(key, value)
	}

	done := make(chan bool, 30)
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := fmt.Sprintf("new-key-%d", n)
			value := []byte(fmt.Sprintf("new-value-%d", n))
			_ = store.Create(key, value)
			done <- true
		}(i)
		go func(n int) {
			key := fmt.Sprintf("key-%d", n)
			value := []byte(fmt.Sprintf("updated-value-%d", n))
			_ = store.Update(key, value)
			done <- true
		}(i)
		go func(n int) {
			key := fmt.Sprintf("key-%d", n+5)
			_ = store.Delete(key)
			done <- true
		}(i)
	}

	for i := 0; i < 30; i++ {
		<-done
	}

	list := store.List()
	if len(list) < 0 {
		t.Error("Store should have some manifests after concurrent operations")
	}
}

func TestManifestStore_IndexSynchronization(t *testing.T) {
	db, err := database.NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	idx := index.NewIndex()
	logger := logr.Discard()
	store := NewManifestStore(db, idx, logger)

	key := "test-key"
	value := []byte("test-value")

	err = store.Create(key, value)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	indexValue, indexOk := store.Get(key)
	dbValue, dbErr := db.Get(key)

	if !indexOk {
		t.Error("Index should have the key")
	}
	if dbErr != nil {
		t.Errorf("DB should have the key: %v", dbErr)
	}
	if string(indexValue) != string(dbValue) {
		t.Errorf("Index and DB values should match: index=%s, db=%s", string(indexValue), string(dbValue))
	}

	updatedValue := []byte("updated-value")
	err = store.Update(key, updatedValue)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	indexValue, indexOk = store.Get(key)
	dbValue, dbErr = db.Get(key)

	if !indexOk {
		t.Error("Index should have the updated key")
	}
	if dbErr != nil {
		t.Errorf("DB should have the updated key: %v", dbErr)
	}
	if string(indexValue) != string(updatedValue) || string(dbValue) != string(updatedValue) {
		t.Errorf("Index and DB should have updated value: index=%s, db=%s", string(indexValue), string(dbValue))
	}

	err = store.Delete(key)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	_, indexOk = store.Get(key)
	_, dbErr = db.Get(key)

	if indexOk {
		t.Error("Index should not have the deleted key")
	}
	if dbErr == nil {
		t.Error("DB should not have the deleted key")
	}
}

