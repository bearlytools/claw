package client

import (
	"iter"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// RecvClient is a client for receive-only streaming RPCs.
// The server sends messages to the client but the client does not send.
type RecvClient struct {
	conn      *Conn
	sessionID uint32
	session   *session
	mu        sync.Mutex
	closed    bool
	err       error
}

// Recv returns an iterator over received payloads.
// The iterator stops on EndStream, Close, or error.
// Call Err() after iteration to check for errors.
func (r *RecvClient) Recv(ctx context.Context) iter.Seq[[]byte] {
	return recvIter(ctx, r.session, &r.err)
}

// Err returns any error that occurred during receiving.
func (r *RecvClient) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// Cancel cancels the RPC, sending a Cancel message to the server.
// This signals to the server that the client is no longer interested
// in receiving more data and the server may stop sending.
// After Cancel, the stream should be closed with Close().
func (r *RecvClient) Cancel() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	return r.conn.sendCancel(r.sessionID, 0)
}

// Close closes the receive client and its session.
func (r *RecvClient) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()

	r.session.close()
	return r.conn.closeSession(r.sessionID, msgs.ErrNone, "")
}
