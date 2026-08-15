package protocol

import (
	"bytes"
	"testing"
)

func TestEncodeDecode_RoundTrip_Store(t *testing.T) {
	original := Message{
		Type: MsgStore,
		Hash: "abc123",
		Data: []byte("some chunk bytes"),
	}

	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, original.Type)
	}
	if decoded.Hash != original.Hash {
		t.Errorf("Hash mismatch: got %s, want %s", decoded.Hash, original.Hash)
	}
	if !bytes.Equal(decoded.Data, original.Data) {
		t.Errorf("Data mismatch: got %v, want %v", decoded.Data, original.Data)
	}
}

func TestEncodeDecode_RoundTrip_HaveNoData(t *testing.T) {
	original := Message{
		Type: MsgHave,
		Hash: "def456",
	}

	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.Type != MsgHave || decoded.Hash != "def456" {
		t.Errorf("unexpected decoded message: %+v", decoded)
	}
	if len(decoded.Data) != 0 {
		t.Errorf("expected empty Data, got %v", decoded.Data)
	}
}

func TestDecode_MultipleMessagesOnSameStream(t *testing.T) {
	// Simulates two messages arriving back-to-back on one TCP connection -
	// a realistic scenario the length-prefix framing must handle correctly.
	msg1, _ := Encode(Message{Type: MsgStore, Hash: "h1", Data: []byte("data1")})
	msg2, _ := Encode(Message{Type: MsgGet, Hash: "h2"})

	stream := bytes.NewReader(append(msg1, msg2...))

	first, err := Decode(stream)
	if err != nil {
		t.Fatalf("Decode first message failed: %v", err)
	}
	if first.Hash != "h1" {
		t.Errorf("expected first message hash 'h1', got %s", first.Hash)
	}

	second, err := Decode(stream)
	if err != nil {
		t.Fatalf("Decode second message failed: %v", err)
	}
	if second.Hash != "h2" {
		t.Errorf("expected second message hash 'h2', got %s", second.Hash)
	}
}

func TestDecode_TruncatedHeaderErrors(t *testing.T) {
	// Only 2 bytes instead of the required 4-byte length header.
	_, err := Decode(bytes.NewReader([]byte{0x00, 0x01}))
	if err == nil {
		t.Fatalf("expected error for truncated header, got nil")
	}
}

func TestDecode_TruncatedPayloadErrors(t *testing.T) {
	full, _ := Encode(Message{Type: MsgStore, Hash: "h1", Data: []byte("full data")})
	// Cut off the payload partway through, keeping the header intact.
	truncated := full[:len(full)-3]

	_, err := Decode(bytes.NewReader(truncated))
	if err == nil {
		t.Fatalf("expected error for truncated payload, got nil")
	}
}
