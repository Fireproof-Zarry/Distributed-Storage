// Package manifest defines the metadata record for an uploaded file: its
// name, size, and the ordered list of chunk hashes needed to reconstruct
// it. A manifest is the "handle" a client needs to download a file later -
// conceptually similar to a root content hash in IPFS.
package manifest

import (
	"encoding/json"
	"fmt"
	"os"

	"godfs/internal/chunk"
)

// Manifest describes how to reassemble one uploaded file from its chunks.
type Manifest struct {
	Filename    string   `json:"filename"`
	Size        int64    `json:"size"`
	ChunkHashes []string `json:"chunk_hashes"`
}

// Build constructs a Manifest from an upload's results.
func Build(filename string, chunkHashes []string, size int64) Manifest {
	return Manifest{
		Filename:    filename,
		Size:        size,
		ChunkHashes: chunkHashes,
	}
}

// Hash returns a content hash for the manifest itself, derived the same way
// chunk hashes are (SHA-256), so a manifest can be identified/verified the
// same way a chunk can.
func (m Manifest) Hash() string {
	data, _ := json.Marshal(m) // Marshal of this struct cannot fail
	return chunk.Hash(data)
}

// Save writes the manifest to disk as indented JSON (human-readable for
// debugging during the MVP).
func Save(m Manifest, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("manifest: failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("manifest: failed to write manifest file %s: %w", path, err)
	}
	return nil
}

// Load reads a manifest back from disk.
func Load(path string) (Manifest, error) {
	var m Manifest

	data, err := os.ReadFile(path)
	if err != nil {
		return m, fmt.Errorf("manifest: failed to read manifest file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("manifest: failed to parse manifest file %s: %w", path, err)
	}
	return m, nil
}
