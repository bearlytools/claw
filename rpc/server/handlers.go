package server

import (
	"fmt"
	"iter"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// Handler is the interface implemented by all RPC type handlers.
type Handler interface {
	Type() msgs.RPCType
}

// SyncHandler handles synchronous request/response RPCs.
type SyncHandler struct {
	HandleFunc func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error)
}

// Type returns the RPC type this handler handles.
func (h SyncHandler) Type() msgs.RPCType {
	return msgs.RTSynchronous
}

// BiDirHandler handles bidirectional streaming RPCs.
type BiDirHandler struct {
	HandleFunc func(ctx context.Context, stream *BiDirStream) error
}

// Type returns the RPC type this handler handles.
func (h BiDirHandler) Type() msgs.RPCType {
	return msgs.RTBiDirectional
}

// SendHandler handles client-send streaming RPCs (client sends, server receives).
type SendHandler struct {
	HandleFunc func(ctx context.Context, stream *RecvStream) error
}

// Type returns the RPC type this handler handles.
func (h SendHandler) Type() msgs.RPCType {
	return msgs.RTSend
}

// RecvHandler handles server-send streaming RPCs (server sends, client receives).
type RecvHandler struct {
	HandleFunc func(ctx context.Context, stream *SendStream) error
}

// Type returns the RPC type this handler handles.
func (h RecvHandler) Type() msgs.RPCType {
	return msgs.RTRecv
}

// BiDirStream provides bidirectional communication for handlers.
type BiDirStream struct {
	sessionID uint32
	conn      *ServerConn
	recvCh    chan msgs.Payload
	cancelCh  chan struct{}
	mu        sync.Mutex
	err       error
	closed    bool
}

func newBiDirStream(sessionID uint32, conn *ServerConn, recvCh chan msgs.Payload, cancelCh chan struct{}) *BiDirStream {
	return &BiDirStream{
		sessionID: sessionID,
		conn:      conn,
		recvCh:    recvCh,
		cancelCh:  cancelCh,
	}
}

// Send sends a payload to the client.
func (s *BiDirStream) Send(payload []byte) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrSessionClosed
	}
	s.mu.Unlock()

	return s.conn.sendPayload(s.sessionID, 0, payload, false)
}

// Recv returns an iterator over received payloads.
func (s *BiDirStream) Recv() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for p := range s.recvCh {
			payload := p.Payload()
			if p.EndStream() && len(payload) == 0 {
				return
			}
			if !yield(payload) {
				return
			}
			if p.EndStream() {
				return
			}
		}
	}
}

// Err returns any error that occurred during the stream.
func (s *BiDirStream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// setErr sets the stream error.
func (s *BiDirStream) setErr(err error) {
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
}

// close marks the stream as closed.
func (s *BiDirStream) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}

// SendStream allows the server to send messages to the client.
// Used by RecvHandler (server sends, client receives).
type SendStream struct {
	sessionID uint32
	conn      *ServerConn
	cancelCh  chan struct{}
	mu        sync.Mutex
	closed    bool
}

func newSendStream(sessionID uint32, conn *ServerConn, cancelCh chan struct{}) *SendStream {
	return &SendStream{
		sessionID: sessionID,
		conn:      conn,
		cancelCh:  cancelCh,
	}
}

// Send sends a payload to the client.
func (s *SendStream) Send(payload []byte) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrSessionClosed
	}
	s.mu.Unlock()

	select {
	case <-s.cancelCh:
		return ErrSessionClosed
	default:
	}

	return s.conn.sendPayload(s.sessionID, 0, payload, false)
}

// close marks the stream as closed.
func (s *SendStream) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}

// RecvStream allows the server to receive messages from the client.
// Used by SendHandler (client sends, server receives).
type RecvStream struct {
	sessionID uint32
	conn      *ServerConn
	recvCh    chan msgs.Payload
	cancelCh  chan struct{}
	mu        sync.Mutex
	err       error
}

func newRecvStream(sessionID uint32, conn *ServerConn, recvCh chan msgs.Payload, cancelCh chan struct{}) *RecvStream {
	return &RecvStream{
		sessionID: sessionID,
		conn:      conn,
		recvCh:    recvCh,
		cancelCh:  cancelCh,
	}
}

// Recv returns an iterator over received payloads.
func (s *RecvStream) Recv() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for p := range s.recvCh {
			payload := p.Payload()
			if p.EndStream() && len(payload) == 0 {
				return
			}
			if !yield(payload) {
				return
			}
			if p.EndStream() {
				return
			}
		}
	}
}

// Err returns any error that occurred during the stream.
func (s *RecvStream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// setErr sets the stream error.
func (s *RecvStream) setErr(err error) {
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
}

// errCodeFromError converts a Go error to an ErrCode.
func errCodeFromError(err error) msgs.ErrCode {
	if err == nil {
		return msgs.ErrNone
	}
	// Default to internal error. Handlers can wrap specific error types
	// to indicate different error codes.
	return msgs.ErrInternal
}

// errorMessage extracts a message from an error.
func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
