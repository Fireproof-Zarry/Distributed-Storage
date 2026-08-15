package client

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"

	"godfs/internal/node"
	"godfs/internal/peer"
)

// startClientTestNode starts a real Node on an OS-assigned port and returns
// its address plus a stop function to shut it down (simulating a node going
// offline, for the fault-tolerance test).
func startClientTestNode(t *testing.T) (addr string, stop func()) {
	t.Helper()

	dir := t.TempDir()
	n, err := node.NewNode("localhost:0", dir)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to bind: %v", err)
	}
	address := ln.Addr().String()
	n.Address = address

	go n.Serve(ln)

	return address, func() { ln.Close() }
}

func writeTestFile(t *testing.T, dir string, sizeBytes int) string {
	t.Helper()
	path := filepath.Join(dir, "testfile.bin")
	data := bytes.Repeat([]byte{0x42}, sizeBytes)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	return path
}

func TestUploadDownload_FullRoundTrip(t *testing.T) {
	// Use 3 nodes so replication (factor 2) leaves genuine redundancy.
	addr1, stop1 := startClientTestNode(t)
	addr2, stop2 := startClientTestNode(t)
	addr3, stop3 := startClientTestNode(t)
	defer stop1()
	defer stop2()
	defer stop3()

	registry := &peer.Registry{Peers: []string{addr1, addr2, addr3}}

	workDir := t.TempDir()
	// ~3 chunks worth of data to exercise multi-chunk upload/download.
	inputPath := writeTestFile(t, workDir, 700*1024)
	manifestPath := filepath.Join(workDir, "manifest.json")
	outPath := filepath.Join(workDir, "output.bin")

	m, err := Upload(inputPath, registry, manifestPath)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if len(m.ChunkHashes) == 0 {
		t.Fatalf("expected at least one chunk hash in manifest")
	}

	if err := Download(manifestPath, registry, outPath); err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	original, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("failed to read original file: %v", err)
	}
	downloaded, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if !bytes.Equal(original, downloaded) {
		t.Fatalf("downloaded file does not match original")
	}
}

func TestDownload_SurvivesOneNodeGoingDown(t *testing.T) {
	// This is the core fault-tolerance guarantee: with replication factor 2
	// and 3 total peers, killing one node must not break downloads, because
	// every chunk still has at least one live replica.
	addr1, stop1 := startClientTestNode(t)
	addr2, stop2 := startClientTestNode(t)
	addr3, stop3 := startClientTestNode(t)
	defer stop2()
	defer stop3()

	registry := &peer.Registry{Peers: []string{addr1, addr2, addr3}}

	workDir := t.TempDir()
	inputPath := writeTestFile(t, workDir, 500*1024)
	manifestPath := filepath.Join(workDir, "manifest.json")
	outPath := filepath.Join(workDir, "output.bin")

	if _, err := Upload(inputPath, registry, manifestPath); err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Simulate node 1 going offline.
	stop1()

	if err := Download(manifestPath, registry, outPath); err != nil {
		t.Fatalf("Download failed after killing one node - replication did not provide fault tolerance: %v", err)
	}

	original, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("failed to read original file: %v", err)
	}
	downloaded, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if !bytes.Equal(original, downloaded) {
		t.Fatalf("downloaded file does not match original after node failure")
	}
}
