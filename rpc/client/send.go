package client

import (
	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// SendClient is a client for send-only streaming RPCs.
// The client sends messages to the server but does not receive responses.
type SendClient struct {
	conn      *Conn
	sessionID uint32
	session   *session
	mu        sync.Mutex
	closed    bool
}

// Send sends a payload to the server.
func (s *SendClient) Send(ctx context.Context, payload []byte) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrSessionClosed
	}
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.conn.closed:
		return ErrClosed
	case <-s.session.cancelCh:
		return ErrSessionClosed
	default:
	}

	return s.conn.sendPayload(s.sessionID, 0, payload, false)
}

// Cancel cancels the RPC, sending a Cancel message to the server.
// This signals to the server that the client is no longer interested
// in the result and the server may stop processing.
// After Cancel, the stream should be closed with Close().
func (s *SendClient) Cancel() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	return s.conn.sendCancel(s.sessionID, 0)
}

// Close closes the send client, signaling end of stream.
func (s *SendClient) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	// Send EndStream to signal we're done.
	s.conn.sendPayload(s.sessionID, 0, nil, true)

	s.session.close()
	return s.conn.closeSession(s.sessionID, msgs.ErrNone, "")
}
