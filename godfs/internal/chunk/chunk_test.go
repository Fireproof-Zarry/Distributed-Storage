package chunk

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// writeTempFile is a small test helper that writes data to a temp file and
// returns its path.
func writeTempFile(t *testing.T, dir string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, "input.bin")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func TestSplitAndReassemble_ExactMultiple(t *testing.T) {
	dir := t.TempDir()
	// Exactly 2 chunks worth of data.
	data := bytes.Repeat([]byte{0xAB}, ChunkSize*2)
	inputPath := writeTempFile(t, dir, data)

	chunks, err := Split(inputPath)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	outPath := filepath.Join(dir, "output.bin")
	if err := Reassemble(chunks, outPath); err != nil {
		t.Fatalf("Reassemble failed: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read reassembled file: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("reassembled data does not match original")
	}
}

func TestSplitAndReassemble_PartialLastChunk(t *testing.T) {
	dir := t.TempDir()
	// 1.5 chunks worth of data - tests the partial last chunk path.
	data := bytes.Repeat([]byte{0xCD}, ChunkSize+ChunkSize/2)
	inputPath := writeTempFile(t, dir, data)

	chunks, err := Split(inputPath)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if len(chunks[1]) != ChunkSize/2 {
		t.Fatalf("expected last chunk to be %d bytes, got %d", ChunkSize/2, len(chunks[1]))
	}

	outPath := filepath.Join(dir, "output.bin")
	if err := Reassemble(chunks, outPath); err != nil {
		t.Fatalf("Reassemble failed: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read reassembled file: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("reassembled data does not match original")
	}
}

func TestHash_Deterministic(t *testing.T) {
	data := []byte("hello godfs")
	h1 := Hash(data)
	h2 := Hash(data)
	if h1 != h2 {
		t.Fatalf("expected same hash for same data, got %s vs %s", h1, h2)
	}
	if len(h1) != 64 { // SHA-256 hex string is always 64 chars
		t.Fatalf("expected 64-char hex hash, got %d chars", len(h1))
	}
}

func TestHash_DifferentDataDifferentHash(t *testing.T) {
	h1 := Hash([]byte("chunk one"))
	h2 := Hash([]byte("chunk two"))
	if h1 == h2 {
		t.Fatalf("expected different hashes for different data")
	}
}

func TestSplit_EmptyFileErrors(t *testing.T) {
	dir := t.TempDir()
	inputPath := writeTempFile(t, dir, []byte{})

	_, err := Split(inputPath)
	if err == nil {
		t.Fatalf("expected error for empty file, got nil")
	}
}

func TestSplit_NonexistentFileErrors(t *testing.T) {
	_, err := Split("/path/does/not/exist.bin")
	if err == nil {
		t.Fatalf("expected error for nonexistent file, got nil")
	}
}
