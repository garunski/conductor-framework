package store

// ManifestStore defines the interface for manifest storage operations.
// This interface allows for better testability and reduced coupling.
type ManifestStore interface {
	// Get retrieves a manifest by key, returning the value and whether it exists
	Get(key string) ([]byte, bool)

	// List returns all manifests as a map of key to value
	List() map[string][]byte

	// Create creates a new manifest entry
	Create(key string, value []byte) error

	// Update updates an existing manifest entry
	Update(key string, value []byte) error

	// Delete deletes a manifest entry by key
	Delete(key string) error
}

// Ensure *manifestStoreImpl implements ManifestStore interface
var _ ManifestStore = (*manifestStoreImpl)(nil)

