package peer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRegistryFile(t *testing.T, dir string, contents string) string {
	t.Helper()
	path := filepath.Join(dir, "peers.json")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write registry file: %v", err)
	}
	return path
}

func TestLoadRegistry_Valid(t *testing.T) {
	dir := t.TempDir()
	path := writeRegistryFile(t, dir, `{"peers": ["localhost:8001", "localhost:8002"]}`)

	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	if len(r.Peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(r.Peers))
	}
}

func TestLoadRegistry_EmptyPeersErrors(t *testing.T) {
	dir := t.TempDir()
	path := writeRegistryFile(t, dir, `{"peers": []}`)

	_, err := LoadRegistry(path)
	if err == nil {
		t.Fatalf("expected error for empty peers list, got nil")
	}
}

func TestLoadRegistry_MissingFileErrors(t *testing.T) {
	_, err := LoadRegistry("/path/does/not/exist.json")
	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}

func TestSelectPeers_Deterministic(t *testing.T) {
	r := &Registry{Peers: []string{"a", "b", "c", "d"}}

	first := r.SelectPeers("somechunkhash", 2)
	second := r.SelectPeers("somechunkhash", 2)

	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("expected 2 peers selected, got %d and %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("expected deterministic selection, got %v vs %v", first, second)
		}
	}
}

func TestSelectPeers_DifferentHashesCanSelectDifferentPeers(t *testing.T) {
	r := &Registry{Peers: []string{"a", "b", "c", "d", "e"}}

	sel1 := r.SelectPeers("hash-one", 2)
	sel2 := r.SelectPeers("hash-two-different", 2)

	// Not a strict guarantee for all inputs, but with 5 peers and these two
	// specific hash strings, selections should differ - guards against a
	// trivially broken (always-same) implementation.
	same := len(sel1) == len(sel2)
	if same {
		for i := range sel1 {
			if sel1[i] != sel2[i] {
				same = false
				break
			}
		}
	}
	if same {
		t.Fatalf("expected different hashes to plausibly select different peers, got identical selections")
	}
}

func TestSelectPeers_NCappedAtPeerCount(t *testing.T) {
	r := &Registry{Peers: []string{"a", "b"}}

	selected := r.SelectPeers("anyhash", 5)
	if len(selected) != 2 {
		t.Fatalf("expected selection capped at 2 peers, got %d", len(selected))
	}
}

func TestSelectPeers_EmptyRegistry(t *testing.T) {
	r := &Registry{Peers: []string{}}
	selected := r.SelectPeers("anyhash", 2)
	if len(selected) != 0 {
		t.Fatalf("expected no peers selected from empty registry, got %d", len(selected))
	}
}
