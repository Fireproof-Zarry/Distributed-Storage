package chunk

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ChunkSize is the fixed size (in bytes) each file chunk is split into.
const ChunkSize = 256 * 1024 // 256 KB

// Split reads the file at filePath and splits it into fixed-size chunks.
// The last chunk may be smaller than ChunkSize.
func Split(filePath string) ([][]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("chunk: failed to open file: %w", err)
	}
	defer f.Close()

	var chunks [][]byte
	buf := make([]byte, ChunkSize)

	for {
		n, err := io.ReadFull(f, buf)
		if n > 0 {
			// Copy since buf is reused across iterations.
			chunkCopy := make([]byte, n)
			copy(chunkCopy, buf[:n])
			chunks = append(chunks, chunkCopy)
		}
		if err == io.EOF {
			break
		}
		if err == io.ErrUnexpectedEOF {
			// Last partial chunk was already captured above.
			break
		}
		if err != nil {
			return nil, fmt.Errorf("chunk: failed reading file: %w", err)
		}
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("chunk: file is empty: %s", filePath)
	}

	return chunks, nil
}

// Hash returns the hex-encoded SHA-256 hash of the given data.
// This is the content-address used to identify a chunk anywhere in the network.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Reassemble writes the given chunks, in order, to outPath as a single file.
func Reassemble(chunks [][]byte, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("chunk: failed to create output file: %w", err)
	}
	defer f.Close()

	for i, c := range chunks {
		if _, err := f.Write(c); err != nil {
			return fmt.Errorf("chunk: failed writing chunk %d: %w", i, err)
		}
	}
	return nil
}
