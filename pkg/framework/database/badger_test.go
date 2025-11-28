package database

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
)

func TestDBGetSet(t *testing.T) {
	db, err := NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}

	testKey := "default/Deployment/test"
	testValue := []byte("test value")

	err = db.Set(testKey, testValue)
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	val, err := db.Get(testKey)
	if err != nil {
		t.Fatalf("failed to get value: %v", err)
	}

	if string(val) != string(testValue) {
		t.Errorf("expected %s, got %s", string(testValue), string(val))
	}
}

func TestDBGetNotFound(t *testing.T) {
	db, err := NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}

	testKey := "default/Deployment/nonexistent"

	_, err = db.Get(testKey)
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Errorf("expected not found error, got %v", err)
	}
}

func TestDBDelete(t *testing.T) {
	db, err := NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}

	testKey := "default/Deployment/test"
	testValue := []byte("test value")

	err = db.Set(testKey, testValue)
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	err = db.Delete(testKey)
	if err != nil {
		t.Fatalf("failed to delete value: %v", err)
	}

	_, err = db.Get(testKey)
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Errorf("expected not found error after delete, got %v", err)
	}
}

func TestDBList(t *testing.T) {
	db, err := NewTestDB(t)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}

	testKey1 := "default/Deployment/test1"
	testValue1 := []byte("test value 1")
	testKey2 := "default/Service/test2"
	testValue2 := []byte("test value 2")
	testKey3 := "kube-system/ConfigMap/test3"
	testValue3 := []byte("test value 3")

	db.Set(testKey1, testValue1)
	db.Set(testKey2, testValue2)
	db.Set(testKey3, testValue3)

	all, err := db.List("")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 items, got %d", len(all))
	}

	defaultItems, err := db.List("default/")
	if err != nil {
		t.Fatalf("failed to list with prefix: %v", err)
	}
	if len(defaultItems) != 2 {
		t.Errorf("expected 2 items with prefix, got %d", len(defaultItems))
	}
}

func TestDBPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-db")
	logger := logr.Discard()

	testKey := "default/Deployment/test"
	testValue := []byte("test value")

	db1, err := NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("failed to create DB: %v", err)
	}
	err = db1.Set(testKey, testValue)
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}
	db1.Close()

	db2, err := NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("failed to reopen DB: %v", err)
	}
	defer db2.Close()

	val, err := db2.Get(testKey)
	if err != nil {
		t.Fatalf("failed to get value after reopen: %v", err)
	}

	if string(val) != string(testValue) {
		t.Errorf("expected %s, got %s", string(testValue), string(val))
	}
}

func TestNewDBCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "new-dir", "test-db")
	logger := logr.Discard()

	db, err := NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("failed to create DB: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		t.Errorf("directory was not created: %v", err)
	}
}

