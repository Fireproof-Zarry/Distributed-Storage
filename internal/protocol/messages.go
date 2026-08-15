package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// MessageType identifies what kind of P2P protocol message this is.
type MessageType string

const (
	MsgStore    MessageType = "STORE"     // request: store a chunk
	MsgGet      MessageType = "GET"       // request: fetch a chunk by hash
	MsgHave     MessageType = "HAVE"      // request: does this node have chunk X?
	MsgAck      MessageType = "ACK"       // response: success (STORE ack, or HAVE=true)
	MsgNack     MessageType = "NACK"      // response: failure / not found (or HAVE=false)
	MsgGetReply MessageType = "GET_REPLY" // response: chunk data for a GET request
)

// Message is the single envelope type used for every request/response
// exchanged between nodes.
type Message struct {
	Type MessageType `json:"type"`
	Hash string      `json:"hash"`
	Data []byte      `json:"data,omitempty"` // present for STORE requests and GET_REPLY responses
}

// maxMessageSize guards against a malformed/malicious length header causing
// an unbounded allocation. 64MB is comfortably above ChunkSize (256KB) plus
// JSON/envelope overhead.
const maxMessageSize = 64 * 1024 * 1024

// Encode serializes a Message as length-prefixed JSON:
// [4 bytes big-endian length][that many bytes of JSON].
func Encode(msg Message) ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("protocol: failed to marshal message: %w", err)
	}

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))

	return append(header, payload...), nil
}

// Decode reads a single length-prefixed JSON message from r.
func Decode(r io.Reader) (Message, error) {
	var msg Message

	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return msg, fmt.Errorf("protocol: failed to read length header: %w", err)
	}

	length := binary.BigEndian.Uint32(header)
	if length > maxMessageSize {
		return msg, fmt.Errorf("protocol: message size %d exceeds max %d", length, maxMessageSize)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return msg, fmt.Errorf("protocol: failed to read payload: %w", err)
	}

	if err := json.Unmarshal(payload, &msg); err != nil {
		return msg, fmt.Errorf("protocol: failed to unmarshal message: %w", err)
	}

	return msg, nil
}
