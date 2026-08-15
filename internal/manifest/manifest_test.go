package manifest

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuild(t *testing.T) {
	m := Build("file.txt", []string{"h1", "h2"}, 1024)

	if m.Filename != "file.txt" {
		t.Errorf("expected filename 'file.txt', got %s", m.Filename)
	}
	if m.Size != 1024 {
		t.Errorf("expected size 1024, got %d", m.Size)
	}
	if !reflect.DeepEqual(m.ChunkHashes, []string{"h1", "h2"}) {
		t.Errorf("unexpected chunk hashes: %v", m.ChunkHashes)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	original := Build("myfile.bin", []string{"hash1", "hash2", "hash3"}, 2048)

	if err := Save(original, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !reflect.DeepEqual(original, loaded) {
		t.Fatalf("loaded manifest does not match original: got %+v, want %+v", loaded, original)
	}
}

func TestHash_Deterministic(t *testing.T) {
	m := Build("file.txt", []string{"h1", "h2"}, 100)

	h1 := m.Hash()
	h2 := m.Hash()
	if h1 != h2 {
		t.Fatalf("expected deterministic hash, got %s vs %s", h1, h2)
	}
}

func TestHash_DifferentManifestsDifferentHash(t *testing.T) {
	m1 := Build("file1.txt", []string{"h1"}, 100)
	m2 := Build("file2.txt", []string{"h1"}, 100)

	if m1.Hash() == m2.Hash() {
		t.Fatalf("expected different manifests to have different hashes")
	}
}

func TestLoad_MissingFileErrors(t *testing.T) {
	_, err := Load("/path/does/not/exist.json")
	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}
