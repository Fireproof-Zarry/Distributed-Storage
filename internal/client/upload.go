// Package client implements the orchestration logic for uploading and
// downloading files against the P2P network: chunking, talking to peers
// over the protocol package, and building/consuming manifests. This is
// deliberately a thin "glue" layer - each thing it calls (chunk, peer,
// protocol, manifest) is independently tested on its own.
package client

import (
	"fmt"
	"path/filepath"

	"godfs/internal/chunk"
	"godfs/internal/manifest"
	"godfs/internal/peer"
	"godfs/internal/protocol"
)

// ReplicationFactor is how many distinct peers each chunk is stored on.
// MVP target: 2 (see design doc Section 6.1).
const ReplicationFactor = 2

// Upload splits the file at filePath into chunks, stores each chunk on
// ReplicationFactor peers selected from the registry, builds a manifest
// describing the file, and saves the manifest to manifestOutPath.
func Upload(filePath string, registry *peer.Registry, manifestOutPath string) (manifest.Manifest, error) {
	var m manifest.Manifest

	chunks, err := chunk.Split(filePath)
	if err != nil {
		return m, fmt.Errorf("upload: failed to split file: %w", err)
	}

	var totalSize int64
	chunkHashes := make([]string, 0, len(chunks))

	for i, c := range chunks {
		hash := chunk.Hash(c)
		chunkHashes = append(chunkHashes, hash)
		totalSize += int64(len(c))

		stored, err := storeChunkOnPeers(hash, c, registry)
		if err != nil {
			return m, fmt.Errorf("upload: chunk %d/%d (%s): %w", i+1, len(chunks), hash, err)
		}
		fmt.Printf("upload: chunk %d/%d (%s) stored on %d/%d target peers\n",
			i+1, len(chunks), hash, stored, ReplicationFactor)
	}

	m = manifest.Build(filepath.Base(filePath), chunkHashes, totalSize)
	if err := manifest.Save(m, manifestOutPath); err != nil {
		return m, fmt.Errorf("upload: failed to save manifest: %w", err)
	}

	return m, nil
}

// storeChunkOnPeers sends a STORE request for one chunk to its selected
// replica peers, and returns how many of them acknowledged successfully.
// It only errors if the chunk could not be stored on ANY peer - partial
// replication (e.g. 1 of 2 peers succeeding) is logged but not fatal, since
// the goal is best-effort replication, not all-or-nothing consistency.
func storeChunkOnPeers(hash string, data []byte, registry *peer.Registry) (int, error) {
	peers := registry.SelectPeers(hash, ReplicationFactor)
	if len(peers) == 0 {
		return 0, fmt.Errorf("no peers available")
	}

	stored := 0
	for _, addr := range peers {
		resp, err := protocol.Send(addr, protocol.Message{
			Type: protocol.MsgStore,
			Hash: hash,
			Data: data,
		})
		if err != nil {
			fmt.Printf("upload: warning: could not reach peer %s: %v\n", addr, err)
			continue
		}
		if resp.Type == protocol.MsgAck {
			stored++
		} else {
			fmt.Printf("upload: warning: peer %s rejected chunk %s\n", addr, hash)
		}
	}

	if stored == 0 {
		return 0, fmt.Errorf("failed to store on any of %d selected peers", len(peers))
	}
	return stored, nil
}
