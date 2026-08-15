// Package node implements a node's local chunk storage.
//
// Day 1 scope: pure disk I/O only (SaveChunk/LoadChunk/HasChunk).
// Networking (Listen/handleConn, turning this into an actual TCP server
// that speaks the protocol package's messages) is added on Day 2 and will
// live alongside this file in the same package.
package node

import (
	"fmt"
	"os"
	"path/filepath"
)

// Store manages a single node's on-disk chunk storage.
// Each chunk is stored as an individual file named by its content hash,
// so a node's DataDir can be inspected directly (e.g. `ls`) to see exactly
// which chunks it holds.
type Store struct {
	DataDir string
}

// NewStore creates a Store rooted at dataDir, creating the directory if it
// doesn't already exist.
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("node: failed to create data dir %s: %w", dataDir, err)
	}
	return &Store{DataDir: dataDir}, nil
}

// chunkPath returns the on-disk path for a chunk with the given hash.
func (s *Store) chunkPath(hash string) string {
	return filepath.Join(s.DataDir, hash)
}

// SaveChunk writes chunk data to disk under its content hash.
// Saving is idempotent: saving the same hash twice with the same data is a
// no-op on the second call (content-addressed storage is naturally
// deduplicating).
func (s *Store) SaveChunk(hash string, data []byte) error {
	path := s.chunkPath(hash)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("node: failed to save chunk %s: %w", hash, err)
	}
	return nil
}

// LoadChunk reads chunk data from disk by its hash.
// The second return value is false if the chunk is not present.
func (s *Store) LoadChunk(hash string) ([]byte, bool) {
	data, err := os.ReadFile(s.chunkPath(hash))
	if err != nil {
		return nil, false
	}
	return data, true
}

// HasChunk reports whether the node currently stores a chunk with the given
// hash, without reading its full contents.
func (s *Store) HasChunk(hash string) bool {
	_, err := os.Stat(s.chunkPath(hash))
	return err == nil
}
