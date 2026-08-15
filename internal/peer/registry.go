// Package peer manages the set of known node addresses a client or node can
// talk to. For the MVP this is a static, hardcoded list loaded from a JSON
// file - a stand-in for a real peer-discovery mechanism (e.g. a Kademlia
// DHT), which is out of scope until Future Scope.
package peer

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
)

// Registry holds the list of known peer addresses.
type Registry struct {
	Peers []string `json:"peers"`
}

// LoadRegistry reads a JSON file of the form {"peers": ["host:port", ...]}.
func LoadRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("peer: failed to read registry file %s: %w", path, err)
	}

	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("peer: failed to parse registry file %s: %w", path, err)
	}
	if len(r.Peers) == 0 {
		return nil, fmt.Errorf("peer: registry file %s has no peers", path)
	}

	return &r, nil
}

// SelectPeers deterministically returns up to n peer addresses for a given
// chunk hash. Determinism matters: it lets a downloader independently
// recompute the same candidate peers an uploader chose, without needing a
// shared lookup table - a small taste of what a real DHT provides.
func (r *Registry) SelectPeers(chunkHash string, n int) []string {
	if len(r.Peers) == 0 {
		return nil
	}
	if n > len(r.Peers) {
		n = len(r.Peers)
	}

	start := hashToIndex(chunkHash, len(r.Peers))
	selected := make([]string, 0, n)
	for i := 0; i < n; i++ {
		idx := (start + i) % len(r.Peers)
		selected = append(selected, r.Peers[idx])
	}
	return selected
}

// hashToIndex maps a hex hash string to an index in [0, mod) using FNV-1a.
// Not cryptographically meaningful - just needs to be deterministic and
// spread chunks reasonably evenly across peers, which a proper hash
// function does far better than summing byte values.
func hashToIndex(hash string, mod int) int {
	if mod == 0 {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(hash))
	return int(h.Sum32()) % mod
}
