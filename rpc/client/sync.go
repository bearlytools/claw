package client

import (
	"fmt"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
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
}

// Call sends a request and blocks until a response or error.
func (s *SyncClient) Call(ctx context.Context, req []byte) ([]byte, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrSessionClosed
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
			return nil, ErrClosed
		case <-s.session.cancelCh:
			return nil, ErrSessionClosed
		case cl := <-s.session.closeCh:
			if cl.ErrCode() != msgs.ErrNone {
				return nil, fmt.Errorf("session closed with error: %s", cl.Error())
			}
			return nil, ErrSessionClosed
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
