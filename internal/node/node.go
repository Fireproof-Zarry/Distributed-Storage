package node

import (
	"fmt"
	"log"
	"net"

	"godfs/internal/protocol"
)

// Node is a P2P storage node: it wraps a local Store and exposes it over
// TCP so other nodes (or the CLI client) can STORE/GET/HAVE? chunks on it.
// Every node runs the exact same code - there is no special "master" node,
// which is the defining property of a P2P (as opposed to client-server)
// architecture.
type Node struct {
	Address string
	Store   *Store
}

// NewNode creates a Node backed by a Store rooted at dataDir.
func NewNode(address, dataDir string) (*Node, error) {
	store, err := NewStore(dataDir)
	if err != nil {
		return nil, err
	}
	return &Node{Address: address, Store: store}, nil
}

// Listen binds to n.Address and blocks forever, accepting connections and
// handling each in its own goroutine.
func (n *Node) Listen() error {
	ln, err := listenTCP(n.Address)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Printf("node: listening on %s (data dir: %s)", ln.Addr().String(), n.Store.DataDir)
	n.Serve(ln)
	return nil
}

// listenTCP is a small wrapper so callers (including tests) can bind to an
// OS-assigned free port (address "localhost:0") and discover the actual
// port before the accept loop starts.
func listenTCP(address string) (net.Listener, error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("node: failed to listen on %s: %w", address, err)
	}
	return ln, nil
}

// Serve runs the accept loop on an already-bound listener. Exported (rather
// than folded into Listen) so callers can bind first - e.g. to an
// OS-assigned port - discover the resulting address, and only then start
// serving. Tests use this to run many nodes on ephemeral ports.
func (n *Node) Serve(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("node: accept error: %v", err)
			return
		}
		go n.handleConn(conn)
	}
}

// handleConn reads exactly one request message, dispatches it based on
// type, and writes back exactly one response message. One goroutine per
// connection keeps this simple - Go's scheduler handles the concurrency.
func (n *Node) handleConn(conn net.Conn) {
	defer conn.Close()

	msg, err := protocol.Decode(conn)
	if err != nil {
		log.Printf("node: failed to decode message from %s: %v", conn.RemoteAddr(), err)
		return
	}

	var resp protocol.Message

	switch msg.Type {
	case protocol.MsgStore:
		if err := n.Store.SaveChunk(msg.Hash, msg.Data); err != nil {
			log.Printf("node: failed to save chunk %s: %v", msg.Hash, err)
			resp = protocol.Message{Type: protocol.MsgNack, Hash: msg.Hash}
		} else {
			log.Printf("node: stored chunk %s (%d bytes)", msg.Hash, len(msg.Data))
			resp = protocol.Message{Type: protocol.MsgAck, Hash: msg.Hash}
		}

	case protocol.MsgHave:
		if n.Store.HasChunk(msg.Hash) {
			resp = protocol.Message{Type: protocol.MsgAck, Hash: msg.Hash}
		} else {
			resp = protocol.Message{Type: protocol.MsgNack, Hash: msg.Hash}
		}

	case protocol.MsgGet:
		data, ok := n.Store.LoadChunk(msg.Hash)
		if !ok {
			resp = protocol.Message{Type: protocol.MsgNack, Hash: msg.Hash}
		} else {
			resp = protocol.Message{Type: protocol.MsgGetReply, Hash: msg.Hash, Data: data}
		}

	default:
		log.Printf("node: unknown message type %q from %s", msg.Type, conn.RemoteAddr())
		resp = protocol.Message{Type: protocol.MsgNack, Hash: msg.Hash}
	}

	encoded, err := protocol.Encode(resp)
	if err != nil {
		log.Printf("node: failed to encode response: %v", err)
		return
	}
	if _, err := conn.Write(encoded); err != nil {
		log.Printf("node: failed to write response to %s: %v", conn.RemoteAddr(), err)
	}
}
