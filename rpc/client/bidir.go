package client

import (
	"iter"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// BiDirClient is a client for bidirectional streaming RPCs.
type BiDirClient struct {
	conn      *Conn
	sessionID uint32
	session   *session
	mu        sync.Mutex
	closed    bool
	sendDone  bool
	err       error
}

// Send sends a payload to the server.
func (b *BiDirClient) Send(ctx context.Context, payload []byte) error {
	b.mu.Lock()
	if b.closed || b.sendDone {
		b.mu.Unlock()
		return ErrSessionClosed
	}
	b.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-b.conn.closed:
		return ErrClosed
	case <-b.session.cancelCh:
		return ErrSessionClosed
	default:
	}

	return b.conn.sendPayload(b.sessionID, 0, payload, false)
}

// Recv returns an iterator over received payloads.
// The iterator stops on EndStream, Close, or error.
// Call Err() after iteration to check for errors.
func (b *BiDirClient) Recv(ctx context.Context) iter.Seq[[]byte] {
	return recvIter(ctx, b.session, &b.err)
}

// Err returns any error that occurred during receiving.
func (b *BiDirClient) Err() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.err
}

// CloseSend signals that the client is done sending.
// The client can still receive messages after this.
func (b *BiDirClient) CloseSend() error {
	b.mu.Lock()
	if b.closed || b.sendDone {
		b.mu.Unlock()
		return nil
	}
	b.sendDone = true
	b.mu.Unlock()

	// Send an empty payload with EndStream=true.
	return b.conn.sendPayload(b.sessionID, 0, nil, true)
}

// Cancel cancels the RPC, sending a Cancel message to the server.
// This signals to the server that the client is no longer interested
// in the result and the server may stop processing.
// After Cancel, the stream should be closed with Close().
func (b *BiDirClient) Cancel() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	return b.conn.sendCancel(b.sessionID, 0)
}

// Close closes the bidirectional client and its session.
func (b *BiDirClient) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	b.session.close()
	return b.conn.closeSession(b.sessionID, msgs.ErrNone, "")
}
