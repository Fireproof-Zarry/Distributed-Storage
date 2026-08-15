package node

import (
	"bytes"
	"testing"
)

func TestSaveAndLoadChunk(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	hash := "somehash123"
	data := []byte("chunk contents here")

	if err := store.SaveChunk(hash, data); err != nil {
		t.Fatalf("SaveChunk failed: %v", err)
	}

	got, ok := store.LoadChunk(hash)
	if !ok {
		t.Fatalf("expected chunk to be found after saving")
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("loaded data does not match saved data")
	}
}

func TestLoadChunk_NotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	_, ok := store.LoadChunk("nonexistent")
	if ok {
		t.Fatalf("expected chunk not to be found")
	}
}

func TestHasChunk(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	hash := "checkme"
	if store.HasChunk(hash) {
		t.Fatalf("expected HasChunk to be false before saving")
	}

	if err := store.SaveChunk(hash, []byte("data")); err != nil {
		t.Fatalf("SaveChunk failed: %v", err)
	}

	if !store.HasChunk(hash) {
		t.Fatalf("expected HasChunk to be true after saving")
	}
}

func TestSaveChunk_Idempotent(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	hash := "dupehash"
	data := []byte("same content")

	if err := store.SaveChunk(hash, data); err != nil {
		t.Fatalf("first SaveChunk failed: %v", err)
	}
	if err := store.SaveChunk(hash, data); err != nil {
		t.Fatalf("second SaveChunk failed: %v", err)
	}

	got, ok := store.LoadChunk(hash)
	if !ok || !bytes.Equal(got, data) {
		t.Fatalf("chunk data corrupted after duplicate save")
	}
}
