package store

import (
	"errors"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/garunski/conductor-framework/pkg/framework/database"
	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"github.com/garunski/conductor-framework/pkg/framework/index"
)

type ManifestStore struct {
	db     *database.DB
	index  *index.ManifestIndex
	logger logr.Logger
}

func NewManifestStore(db *database.DB, idx *index.ManifestIndex, logger logr.Logger) *ManifestStore {
	return &ManifestStore{
		db:     db,
		index:  idx,
		logger: logger,
	}
}

func (s *ManifestStore) Create(key string, value []byte) error {
	if err := s.db.Set(key, value); err != nil {
		return fmt.Errorf("db set: %w", err)
	}
	s.index.Set(key, value)
	return nil
}

func (s *ManifestStore) Update(key string, value []byte) error {

	if _, exists := s.index.Get(key); !exists {
		return fmt.Errorf("%w: manifest not found: %s", apperrors.ErrNotFound, key)
	}

	if err := s.db.Set(key, value); err != nil {
		return fmt.Errorf("db set: %w", err)
	}
	s.index.Set(key, value)
	return nil
}

func (s *ManifestStore) Delete(key string) error {

	if _, exists := s.index.Get(key); !exists {
		return fmt.Errorf("%w: manifest not found: %s", apperrors.ErrNotFound, key)
	}

	if err := s.db.Delete(key); err != nil {

		if !errors.Is(err, database.ErrNotFound) {
			return fmt.Errorf("db delete: %w", err)
		}

		s.logger.Info("key exists in index but not in DB, removing from index", "key", key)
	}

	s.index.Delete(key)
	return nil
}

func (s *ManifestStore) Get(key string) ([]byte, bool) {
	return s.index.Get(key)
}

func (s *ManifestStore) List() map[string][]byte {
	return s.index.List()
}

