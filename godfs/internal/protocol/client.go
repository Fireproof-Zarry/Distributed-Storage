package protocol

import (
	"fmt"
	"net"
	"time"
)

// dialTimeout bounds how long we wait to connect to an unreachable peer -
// important for the fault-tolerance path, where some peers may be down.
const dialTimeout = 3 * time.Second

// Send opens a TCP connection to address, writes msg, waits for exactly one
// response message, and closes the connection. This is the client-side
// counterpart to a Node's handleConn, which reads exactly one request and
// writes exactly one response per connection.
func Send(address string, msg Message) (Message, error) {
	var resp Message

	conn, err := net.DialTimeout("tcp", address, dialTimeout)
	if err != nil {
		return resp, fmt.Errorf("protocol: failed to connect to %s: %w", address, err)
	}
	defer conn.Close()

	encoded, err := Encode(msg)
	if err != nil {
		return resp, err
	}
	if _, err := conn.Write(encoded); err != nil {
		return resp, fmt.Errorf("protocol: failed to send message to %s: %w", address, err)
	}

	resp, err = Decode(conn)
	if err != nil {
		return resp, fmt.Errorf("protocol: failed to read response from %s: %w", address, err)
	}
	return resp, nil
}
