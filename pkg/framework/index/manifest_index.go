package index

import (
	"sync"
)

type ManifestIndex struct {
	mu        sync.RWMutex
	manifests map[string][]byte
}

func NewIndex() *ManifestIndex {
	return &ManifestIndex{
		manifests: make(map[string][]byte),
	}
}

func copyBytes(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func (idx *ManifestIndex) Get(key string) ([]byte, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	val, ok := idx.manifests[key]
	if !ok {
		return nil, false
	}

	return copyBytes(val), true
}

func (idx *ManifestIndex) Set(key string, value []byte) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.manifests[key] = copyBytes(value)
}

func (idx *ManifestIndex) Delete(key string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	delete(idx.manifests, key)
}

func (idx *ManifestIndex) List() map[string][]byte {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make(map[string][]byte)
	for k, v := range idx.manifests {
		result[k] = copyBytes(v)
	}
	return result
}

func (idx *ManifestIndex) Merge(embedded map[string][]byte, dbOverrides map[string][]byte) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.manifests = make(map[string][]byte)
	for k, v := range embedded {
		idx.manifests[k] = copyBytes(v)
	}

	for k, v := range dbOverrides {
		idx.manifests[k] = copyBytes(v)
	}
}

