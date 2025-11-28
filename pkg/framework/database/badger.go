package database

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/go-logr/logr"
	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
)

var ErrNotFound = errors.New("key not found")

type DB struct {
	db     *badger.DB
	logger logr.Logger
}

func NewDB(path string, logger logr.Logger) (*DB, error) {

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("%w: storage create directory: failed to create DB directory at %s: %w", apperrors.ErrStorage, path, err)
	}

	opts := badger.DefaultOptions(path)
	opts.Logger = nil

	opts.ValueLogFileSize = 1 << 30

	opts.NumMemtables = 5

	opts.NumLevelZeroTables = 5

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("%w: storage open database: failed to open BadgerDB at %s: %w", apperrors.ErrStorage, path, err)
	}

	return &DB{
		db:     db,
		logger: logger,
	}, nil
}

func (d *DB) Get(key string) ([]byte, error) {
	var value []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			value = append([]byte{}, val...)
			return nil
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, fmt.Errorf("key not found: %s: %w", key, ErrNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: storage get %s: %w", apperrors.ErrStorage, key, err)
	}

	return value, nil
}

func (d *DB) update(operation string, key string, fn func(*badger.Txn) error) error {
	err := d.db.Update(fn)
	if err != nil {
		return fmt.Errorf("%w: storage %s %s: %w", apperrors.ErrStorage, operation, key, err)
	}
	return nil
}

func (d *DB) Set(key string, value []byte) error {
	return d.update("set", key, func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (d *DB) Delete(key string) error {
	return d.update("delete", key, func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (d *DB) List(prefix string) (map[string][]byte, error) {
	results := make(map[string][]byte)
	err := d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			err := item.Value(func(val []byte) error {
				results[key] = append([]byte{}, val...)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: storage list %s: %w", apperrors.ErrStorage, prefix, err)
	}
	return results, nil
}

func (d *DB) BatchSet(items map[string][]byte) error {
	txn := d.db.NewTransaction(true)
	defer txn.Discard()

	for key, value := range items {
		if err := txn.Set([]byte(key), value); err != nil {
			return fmt.Errorf("%w: storage batch set %s: %w", apperrors.ErrStorage, key, err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("%w: storage batch set commit: %w", apperrors.ErrStorage, err)
	}

	return nil
}

func (d *DB) BatchDelete(keys []string) error {
	txn := d.db.NewTransaction(true)
	defer txn.Discard()

	for _, key := range keys {
		if err := txn.Delete([]byte(key)); err != nil {

			d.logger.V(1).Info("failed to delete key in batch", "key", key, "error", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("%w: storage batch delete commit: %w", apperrors.ErrStorage, err)
	}

	return nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

// NewTestDB creates a test database for testing purposes
func NewTestDB(t testing.TB) (*DB, error) {
	opts := badger.DefaultOptions("").WithInMemory(true)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create test DB: %w", err)
	}
	testDB := &DB{db: db, logger: logr.Discard()}
	if t != nil {
		if cleanup, ok := t.(interface{ Cleanup(func()) }); ok {
			cleanup.Cleanup(func() { testDB.Close() })
		}
	}
	return testDB, nil
}

