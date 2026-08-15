package client

import (
	"fmt"

	"godfs/internal/chunk"
	"godfs/internal/manifest"
	"godfs/internal/peer"
	"godfs/internal/protocol"
)

// Download loads the manifest at manifestPath, fetches each chunk from
// whichever known peer has it, verifies its content hash, and reassembles
// the original file at outPath.
func Download(manifestPath string, registry *peer.Registry, outPath string) error {
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return fmt.Errorf("download: failed to load manifest: %w", err)
	}

	chunks := make([][]byte, 0, len(m.ChunkHashes))

	for i, hash := range m.ChunkHashes {
		data, err := fetchChunk(hash, registry)
		if err != nil {
			return fmt.Errorf("download: chunk %d/%d (%s): %w", i+1, len(m.ChunkHashes), hash, err)
		}

		// Content-addressing payoff: verify locally, don't trust the peer.
		if chunk.Hash(data) != hash {
			return fmt.Errorf("download: chunk %d/%d (%s) failed integrity check", i+1, len(m.ChunkHashes), hash)
		}

		chunks = append(chunks, data)
		fmt.Printf("download: chunk %d/%d (%s) fetched and verified\n", i+1, len(m.ChunkHashes), hash)
	}

	if err := chunk.Reassemble(chunks, outPath); err != nil {
		return fmt.Errorf("download: failed to reassemble file: %w", err)
	}

	return nil
}

// fetchChunk tries each candidate peer for the given hash, in the same
// deterministic order Upload used to place replicas, until one returns the
// data. If none of the deterministic candidates respond (e.g. a node is
// down), it falls back to trying every known peer - this is what makes the
// "kill a node, download still works" demo possible.
func fetchChunk(hash string, registry *peer.Registry) ([]byte, error) {
	var lastErr error

	for _, addr := range registry.SelectPeers(hash, ReplicationFactor) {
		if data, err := tryFetchFrom(addr, hash); err == nil {
			return data, nil
		} else {
			lastErr = err
		}
	}

	// Fallback: try every known peer, not just the deterministic replica
	// set, in case placement and the current peer list have diverged.
	for _, addr := range registry.Peers {
		if data, err := tryFetchFrom(addr, hash); err == nil {
			return data, nil
		} else {
			lastErr = err
		}
	}

	return nil, fmt.Errorf("not found on any reachable peer (last error: %v)", lastErr)
}

func tryFetchFrom(addr, hash string) ([]byte, error) {
	resp, err := protocol.Send(addr, protocol.Message{Type: protocol.MsgGet, Hash: hash})
	if err != nil {
		return nil, err
	}
	if resp.Type != protocol.MsgGetReply {
		return nil, fmt.Errorf("peer %s does not have chunk", addr)
	}
	return resp.Data, nil
}
