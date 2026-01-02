package client

import (
	"fmt"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/errors"
	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/metadata"
)

// SyncClient is a client for synchronous request/response RPCs.
type SyncClient struct {
	conn      *Conn
	sessionID uint32
	session   *session
	nextReqID uint32
	mu        sync.Mutex
	pending   map[uint32]chan response
	closed    bool
	method    string // "pkg/service/call" for interceptors
}

// Call sends a request and blocks until a response or error.
func (s *SyncClient) Call(ctx context.Context, req []byte) ([]byte, error) {
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		return s.doCall(ctx, req)
	}

	if s.conn.unaryInterceptor != nil {
		return s.conn.unaryInterceptor(ctx, s.method, req, invoker)
	}
	return invoker(ctx, req)
}

// doCall performs the actual RPC call.
func (s *SyncClient) doCall(ctx context.Context, req []byte) ([]byte, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, errors.E(ctx, errors.Unavailable, ErrSessionClosed)
	}

	reqID := s.nextReqID
	s.nextReqID++

	respCh := make(chan response, 1)
	s.pending[reqID] = respCh
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, reqID)
		s.mu.Unlock()
	}()

	// Send the request.
	if err := s.conn.sendPayload(s.sessionID, reqID, req, false); err != nil {
		return nil, err
	}

	// Wait for response.
	for {
		select {
		case <-ctx.Done():
			// Send cancel.
			s.conn.sendCancel(s.sessionID, reqID)
			return nil, ctx.Err()
		case <-s.conn.closed:
			return nil, errors.E(ctx, errors.Unavailable, ErrClosed)
		case <-s.session.cancelCh:
			return nil, errors.E(ctx, errors.Unavailable, ErrSessionClosed)
		case cl := <-s.session.closeCh:
			if cl.ErrCode() != msgs.ErrNone {
				return nil, errors.E(ctx, errors.Category(cl.ErrCode()), fmt.Errorf("server error: %s", cl.Error()))
			}
			return nil, errors.E(ctx, errors.Unavailable, ErrSessionClosed)
		case p := <-s.session.recvCh:
			if p.ReqID() == reqID {
				return p.Payload(), nil
			}
			// Response for different request, dispatch to appropriate channel.
			s.mu.Lock()
			if ch, ok := s.pending[p.ReqID()]; ok {
				select {
				case ch <- response{payload: p.Payload()}:
				default:
				}
			}
			s.mu.Unlock()
		case resp := <-respCh:
			return resp.payload, resp.err
		}
	}
}

// compile-time check that interceptor is compatible
var _ interceptor.UnaryInvoker = (*SyncClient)(nil).doCall

// Cancel cancels any pending RPC calls on this client.
// This sends a Cancel message to the server for the session.
// Note: Individual calls already cancel automatically when their
// context is cancelled.
func (s *SyncClient) Cancel() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	return s.conn.sendCancel(s.sessionID, 0)
}

// Close closes the sync client and its session.
func (s *SyncClient) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	s.session.close()
	return s.conn.closeSession(s.sessionID, msgs.ErrNone, "")
}

// ResponseMetadata returns the metadata received from the server in OpenAck.
// Returns nil if no metadata was received.
func (s *SyncClient) ResponseMetadata() metadata.MD {
	return s.session.respMD
}
