// Package client provides RPC client functionality for multiplexed connections.
package client

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"time"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// Protocol version constants.
const (
	ProtocolMajor = 1
	ProtocolMinor = 0
)

// Common errors.
var (
	ErrClosed        = errors.New("connection closed")
	ErrSessionClosed = errors.New("session closed")
	ErrFatalError    = errors.New("fatal connection error")
	ErrTimeout       = errors.New("operation timed out")
	ErrCanceled      = errors.New("operation canceled")
)

// Option configures a Conn.
type Option func(*Conn)

// WithPingInterval sets the interval between keepalive pings.
func WithPingInterval(d time.Duration) Option {
	return func(c *Conn) {
		c.pingInterval = d
	}
}

// WithPingTimeout sets the timeout for ping responses.
func WithPingTimeout(d time.Duration) Option {
	return func(c *Conn) {
		c.pingTimeout = d
	}
}

// WithMaxPayloadSize sets the maximum payload size to advertise to the server.
func WithMaxPayloadSize(size uint32) Option {
	return func(c *Conn) {
		c.maxPayloadSize = size
	}
}

// session represents an active RPC session.
type session struct {
	id        uint32
	rpcType   msgs.RPCType
	recvCh    chan msgs.Payload
	closeCh   chan msgs.Close
	cancelCh  chan struct{}
	readyCh   chan struct{} // Closed when session is ready (OpenAck received)
	closeOnce sync.Once
}

func newSession(id uint32, rpcType msgs.RPCType) *session {
	return &session{
		id:       id,
		rpcType:  rpcType,
		recvCh:   make(chan msgs.Payload, 16),
		closeCh:  make(chan msgs.Close, 1),
		cancelCh: make(chan struct{}),
		readyCh:  make(chan struct{}),
	}
}

func (s *session) close() {
	s.closeOnce.Do(func() {
		close(s.cancelCh)
	})
}

// response holds a response for a synchronous call.
type response struct {
	payload []byte
	err     error
}

// Conn manages a single transport connection with multiple muxed sessions.
type Conn struct {
	transport io.ReadWriteCloser

	sessions   map[uint32]*session // SessionID -> session
	pending    map[uint32]*session // OpenID -> session (waiting for OpenAck)
	mu         sync.Mutex
	writeMu    sync.Mutex // Serializes writes to transport
	nextOpenID uint32

	closed   chan struct{}
	fatalErr error

	pingInterval   time.Duration
	pingTimeout    time.Duration
	maxPayloadSize uint32

	ctx context.Context
}

// New creates a new connection over the given transport.
func New(ctx context.Context, transport io.ReadWriteCloser, opts ...Option) *Conn {
	c := &Conn{
		transport:      transport,
		sessions:       make(map[uint32]*session),
		pending:        make(map[uint32]*session),
		closed:         make(chan struct{}),
		pingInterval:   30 * time.Second,
		pingTimeout:    10 * time.Second,
		maxPayloadSize: 4 * 1024 * 1024, // 4MB default
		ctx:            ctx,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Start the read loop in the pool.
	pool := context.Pool(ctx)
	pool.Submit(ctx, func() {
		c.readLoop()
	})

	return c
}

// Close closes the connection and all sessions.
func (c *Conn) Close() error {
	c.mu.Lock()
	select {
	case <-c.closed:
		c.mu.Unlock()
		return nil
	default:
		close(c.closed)
	}

	// Close all sessions.
	for _, sess := range c.sessions {
		sess.close()
	}
	for _, sess := range c.pending {
		sess.close()
	}
	c.mu.Unlock()

	return c.transport.Close()
}

// Err returns any fatal error that occurred on the connection.
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fatalErr
}

// setFatalError sets a fatal error and closes the connection.
func (c *Conn) setFatalError(err error) {
	c.mu.Lock()
	if c.fatalErr == nil {
		c.fatalErr = err
	}
	c.mu.Unlock()
	c.Close()
}

// readLoop reads messages from the transport and dispatches them.
func (c *Conn) readLoop() {
	for {
		select {
		case <-c.closed:
			return
		case <-c.ctx.Done():
			return
		default:
		}

		msg := msgs.NewMsg()
		_, err := msg.UnmarshalReader(c.transport)
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.setFatalError(io.EOF)
				return
			}
			c.setFatalError(fmt.Errorf("read error: %w", err))
			return
		}

		switch msg.Type() {
		case msgs.TOpenAck:
			c.handleOpenAck(msg.OpenAck())
		case msgs.TClose:
			c.handleClose(msg.Close())
		case msgs.TPayload:
			c.handlePayload(msg.Payload())
		case msgs.TPong:
			c.handlePong(msg.Pong())
		case msgs.TGoAway:
			c.handleGoAway(msg.GoAway())
		default:
			// Unknown message type, ignore.
		}
	}
}

// handleOpenAck processes an OpenAck message.
func (c *Conn) handleOpenAck(ack msgs.OpenAck) {
	c.mu.Lock()
	sess, ok := c.pending[ack.OpenID()]
	if !ok {
		c.mu.Unlock()
		return
	}
	delete(c.pending, ack.OpenID())

	if ack.ErrCode() != msgs.ErrNone {
		// Open was rejected.
		close(sess.readyCh) // Signal that session setup is complete (even though it failed)
		sess.close()
		c.mu.Unlock()
		return
	}

	sess.id = ack.SessionID()
	c.sessions[sess.id] = sess
	close(sess.readyCh) // Signal that session is ready
	c.mu.Unlock()
}

// handleClose processes a Close message.
func (c *Conn) handleClose(cl msgs.Close) {
	c.mu.Lock()
	sess, ok := c.sessions[cl.SessionID()]
	if !ok {
		c.mu.Unlock()
		return
	}
	delete(c.sessions, cl.SessionID())
	c.mu.Unlock()

	select {
	case sess.closeCh <- cl:
	default:
	}
	sess.close()
}

// handlePayload processes a Payload message.
func (c *Conn) handlePayload(p msgs.Payload) {
	c.mu.Lock()
	sess, ok := c.sessions[p.SessionID()]
	c.mu.Unlock()
	if !ok {
		return
	}

	select {
	case sess.recvCh <- p:
	case <-sess.cancelCh:
	}
}

// handlePong processes a Pong message.
func (c *Conn) handlePong(p msgs.Pong) {
	// TODO: Track ping/pong for keepalive.
}

// handleGoAway processes a GoAway message.
func (c *Conn) handleGoAway(ga msgs.GoAway) {
	// Server is going away. Mark connection as draining.
	// New sessions after LastSessionID will fail.
	c.setFatalError(fmt.Errorf("server going away: %s", ga.DebugData()))
}

// sendMsg sends a message on the transport.
func (c *Conn) sendMsg(msg msgs.Msg) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	select {
	case <-c.closed:
		return ErrClosed
	default:
	}

	_, err := msg.MarshalWriter(c.transport)
	if err != nil {
		c.setFatalError(fmt.Errorf("write error: %w", err))
		return err
	}
	return nil
}

// openSession opens a new session with the server.
func (c *Conn) openSession(ctx context.Context, pkg, service, call string, rpcType msgs.RPCType) (*session, error) {
	c.mu.Lock()
	select {
	case <-c.closed:
		c.mu.Unlock()
		return nil, ErrClosed
	default:
	}

	openID := c.nextOpenID
	c.nextOpenID++

	sess := newSession(0, rpcType)
	c.pending[openID] = sess
	c.mu.Unlock()

	// Build the Open message.
	descr := msgs.NewDescr().
		SetPackage(pkg).
		SetService(service).
		SetCall(call).
		SetType(rpcType)

	open := msgs.NewOpen().
		SetOpenID(openID).
		SetDescr(descr).
		SetProtocolMajor(ProtocolMajor).
		SetProtocolMinor(ProtocolMinor).
		SetMaxPayloadSize(c.maxPayloadSize)

	// Set deadline if context has one.
	if deadline, ok := ctx.Deadline(); ok {
		ms := time.Until(deadline).Milliseconds()
		if ms > 0 {
			open = open.SetDeadlineMS(uint64(ms))
		}
	}

	msg := msgs.NewMsg().SetType(msgs.TOpen).SetOpen(open)

	if err := c.sendMsg(msg); err != nil {
		c.mu.Lock()
		delete(c.pending, openID)
		c.mu.Unlock()
		return nil, err
	}

	// Wait for OpenAck via the ready channel.
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, openID)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.closed:
		return nil, ErrClosed
	case <-timeout.C:
		c.mu.Lock()
		delete(c.pending, openID)
		c.mu.Unlock()
		return nil, ErrTimeout
	case <-sess.readyCh:
		// OpenAck was received. Check if accepted or rejected.
		// If sess.id != 0, the session was accepted and added to sessions map.
		if sess.id != 0 {
			return sess, nil
		}
		// sess.id == 0 means the open was rejected.
		return nil, errors.New("session open rejected by server")
	}
}

// closeSession sends a Close message for a session.
func (c *Conn) closeSession(sessionID uint32, errCode msgs.ErrCode, errMsg string) error {
	cl := msgs.NewClose().
		SetSessionID(sessionID).
		SetErrCode(errCode).
		SetError(errMsg)

	msg := msgs.NewMsg().SetType(msgs.TClose).SetClose(cl)
	return c.sendMsg(msg)
}

// sendPayload sends a Payload message.
func (c *Conn) sendPayload(sessionID, reqID uint32, payload []byte, endStream bool) error {
	p := msgs.NewPayload().
		SetSessionID(sessionID).
		SetReqID(reqID).
		SetPayload(payload).
		SetEndStream(endStream)

	msg := msgs.NewMsg().SetType(msgs.TPayload).SetPayload(p)
	return c.sendMsg(msg)
}

// sendCancel sends a Cancel message.
func (c *Conn) sendCancel(sessionID, reqID uint32) error {
	cancel := msgs.NewCancel().
		SetSessionID(sessionID).
		SetReqID(reqID)

	msg := msgs.NewMsg().SetType(msgs.TCancel).SetCancel(cancel)
	return c.sendMsg(msg)
}

// Sync creates a new synchronous RPC client.
func (c *Conn) Sync(ctx context.Context, pkg, service, call string) (*SyncClient, error) {
	sess, err := c.openSession(ctx, pkg, service, call, msgs.RTSynchronous)
	if err != nil {
		return nil, err
	}

	return &SyncClient{
		conn:      c,
		sessionID: sess.id,
		session:   sess,
		pending:   make(map[uint32]chan response),
	}, nil
}

// BiDir creates a new bidirectional streaming RPC client.
func (c *Conn) BiDir(ctx context.Context, pkg, service, call string) (*BiDirClient, error) {
	sess, err := c.openSession(ctx, pkg, service, call, msgs.RTBiDirectional)
	if err != nil {
		return nil, err
	}

	return &BiDirClient{
		conn:      c,
		sessionID: sess.id,
		session:   sess,
	}, nil
}

// Send creates a new send-only streaming RPC client.
func (c *Conn) Send(ctx context.Context, pkg, service, call string) (*SendClient, error) {
	sess, err := c.openSession(ctx, pkg, service, call, msgs.RTSend)
	if err != nil {
		return nil, err
	}

	return &SendClient{
		conn:      c,
		sessionID: sess.id,
		session:   sess,
	}, nil
}

// Recv creates a new receive-only streaming RPC client.
func (c *Conn) Recv(ctx context.Context, pkg, service, call string) (*RecvClient, error) {
	sess, err := c.openSession(ctx, pkg, service, call, msgs.RTRecv)
	if err != nil {
		return nil, err
	}

	return &RecvClient{
		conn:      c,
		sessionID: sess.id,
		session:   sess,
	}, nil
}

// recvIter creates an iterator for receiving payloads from a session.
func recvIter(ctx context.Context, sess *session, errPtr *error) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for {
			select {
			case <-ctx.Done():
				*errPtr = ctx.Err()
				return
			case <-sess.cancelCh:
				return
			case cl := <-sess.closeCh:
				if cl.ErrCode() != msgs.ErrNone {
					*errPtr = fmt.Errorf("session closed with error: %s", cl.Error())
				}
				return
			case p := <-sess.recvCh:
				payload := p.Payload()
				endStream := p.EndStream()
				// If EndStream with no payload, just exit without yielding.
				if endStream && len(payload) == 0 {
					return
				}
				if !yield(payload) {
					return
				}
				if endStream {
					return
				}
			}
		}
	}
}
