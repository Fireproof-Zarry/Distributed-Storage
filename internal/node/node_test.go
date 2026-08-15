package node

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"godfs/internal/protocol"
)

// startTestNode starts a Node listening on an OS-assigned free port and
// returns its address once it's ready to accept connections.
func startTestNode(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	n, err := NewNode("localhost:0", dir)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}

	// Bind manually so we can discover the OS-assigned port before Listen's
	// accept loop takes over.
	ln, err := listenTCP(n.Address)
	if err != nil {
		t.Fatalf("failed to bind test listener: %v", err)
	}
	addr := ln.Addr().String()
	n.Address = addr

	go n.Serve(ln)

	return addr
}

func TestNode_StoreThenHaveThenGet(t *testing.T) {
	addr := startTestNode(t)

	hash := "testhash1"
	data := []byte("hello over the network")

	storeResp, err := protocol.Send(addr, protocol.Message{
		Type: protocol.MsgStore,
		Hash: hash,
		Data: data,
	})
	if err != nil {
		t.Fatalf("STORE failed: %v", err)
	}
	if storeResp.Type != protocol.MsgAck {
		t.Fatalf("expected ACK for STORE, got %s", storeResp.Type)
	}

	haveResp, err := protocol.Send(addr, protocol.Message{
		Type: protocol.MsgHave,
		Hash: hash,
	})
	if err != nil {
		t.Fatalf("HAVE failed: %v", err)
	}
	if haveResp.Type != protocol.MsgAck {
		t.Fatalf("expected ACK for HAVE (chunk exists), got %s", haveResp.Type)
	}

	getResp, err := protocol.Send(addr, protocol.Message{
		Type: protocol.MsgGet,
		Hash: hash,
	})
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if getResp.Type != protocol.MsgGetReply {
		t.Fatalf("expected GET_REPLY, got %s", getResp.Type)
	}
	if !bytes.Equal(getResp.Data, data) {
		t.Fatalf("GET returned wrong data: got %v, want %v", getResp.Data, data)
	}
}

func TestNode_HaveUnknownChunkReturnsNack(t *testing.T) {
	addr := startTestNode(t)

	resp, err := protocol.Send(addr, protocol.Message{
		Type: protocol.MsgHave,
		Hash: "never-stored",
	})
	if err != nil {
		t.Fatalf("HAVE failed: %v", err)
	}
	if resp.Type != protocol.MsgNack {
		t.Fatalf("expected NACK for unknown chunk, got %s", resp.Type)
	}
}

func TestNode_GetUnknownChunkReturnsNack(t *testing.T) {
	addr := startTestNode(t)

	resp, err := protocol.Send(addr, protocol.Message{
		Type: protocol.MsgGet,
		Hash: "never-stored",
	})
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if resp.Type != protocol.MsgNack {
		t.Fatalf("expected NACK for unknown chunk, got %s", resp.Type)
	}
}

func TestNode_ConcurrentStores(t *testing.T) {
	addr := startTestNode(t)

	const n = 20
	done := make(chan error, n)

	for i := 0; i < n; i++ {
		go func(i int) {
			hash := "concurrent-hash"
			data := []byte("data")
			_ = i
			resp, err := protocol.Send(addr, protocol.Message{
				Type: protocol.MsgStore,
				Hash: hash,
				Data: data,
			})
			if err != nil {
				done <- err
				return
			}
			if resp.Type != protocol.MsgAck {
				done <- fmt.Errorf("expected ACK, got %s", resp.Type)
				return
			}
			done <- nil
		}(i)
	}

	for i := 0; i < n; i++ {
		if err := <-done; err != nil {
			t.Fatalf("concurrent STORE failed: %v", err)
		}
	}

	time.Sleep(50 * time.Millisecond) // let any stragglers finish
}
