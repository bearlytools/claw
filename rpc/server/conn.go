package server

import (
	"errors"
	"fmt"
	"io"
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

// serverSession represents a server-side session.
type serverSession struct {
	id        uint32
	rpcType   msgs.RPCType
	handler   Handler
	recvCh    chan msgs.Payload
	cancelCh  chan struct{}
	closeOnce sync.Once
	metadata  []msgs.Metadata
}

func newServerSession(id uint32, rpcType msgs.RPCType, handler Handler, md []msgs.Metadata) *serverSession {
	return &serverSession{
		id:       id,
		rpcType:  rpcType,
		handler:  handler,
		recvCh:   make(chan msgs.Payload, 16),
		cancelCh: make(chan struct{}),
		metadata: md,
	}
}

func (s *serverSession) close() {
	s.closeOnce.Do(func() {
		close(s.cancelCh)
	})
}

// ServerConn handles a single client connection.
type ServerConn struct {
	server    *Server
	transport io.ReadWriteCloser
	sessions  map[uint32]*serverSession
	mu        sync.Mutex
	writeMu   sync.Mutex
	closed    chan struct{}
	fatalErr  error

	nextSessionID uint32

	ctx context.Context
}

func newServerConn(ctx context.Context, server *Server, transport io.ReadWriteCloser) *ServerConn {
	return &ServerConn{
		server:        server,
		transport:     transport,
		sessions:      make(map[uint32]*serverSession),
		closed:        make(chan struct{}),
		nextSessionID: 1,
		ctx:           ctx,
	}
}

// serve runs the main read loop for this connection.
func (c *ServerConn) serve(ctx context.Context) error {
	// Close all recvCh when serve exits. This signals handlers to drain and exit.
	// Safe because handlePayload (the only sender) runs in this same goroutine.
	defer func() {
		c.mu.Lock()
		for _, sess := range c.sessions {
			close(sess.recvCh)
		}
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.closed:
			return c.fatalErr
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg := msgs.NewMsg(ctx)
		_, err := msg.UnmarshalReader(c.transport)
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.setFatalError(io.EOF)
				return nil
			}
			c.setFatalError(fmt.Errorf("read error: %w", err))
			return err
		}

		switch msg.Type() {
		case msgs.TOpen:
			c.handleOpen(ctx, msg.Open())
		case msgs.TClose:
			c.handleClose(msg.Close())
		case msgs.TPayload:
			c.handlePayload(msg.Payload())
		case msgs.TCancel:
			c.handleCancel(msg.Cancel())
		case msgs.TPing:
			c.handlePing(msg.Ping())
		default:
			// Unknown message type, ignore.
		}
	}
}

// Close closes the connection.
func (c *ServerConn) Close() error {
	c.mu.Lock()
	select {
	case <-c.closed:
		c.mu.Unlock()
		return nil
	default:
		close(c.closed)
	}

	for _, sess := range c.sessions {
		sess.close()
	}
	c.mu.Unlock()

	return c.transport.Close()
}

// setFatalError sets a fatal error and closes the connection.
func (c *ServerConn) setFatalError(err error) {
	c.mu.Lock()
	if c.fatalErr == nil {
		c.fatalErr = err
	}
	c.mu.Unlock()
	c.Close()
}

// goAway sends a GoAway message to the client.
func (c *ServerConn) goAway(ctx context.Context) {
	c.mu.Lock()
	lastSessionID := c.nextSessionID - 1
	c.mu.Unlock()

	ga := msgs.NewGoAway(ctx).
		SetLastSessionID(lastSessionID).
		SetErrCode(msgs.ErrNone).
		SetDebugData("server shutting down")

	msg := msgs.NewMsg(ctx).SetType(msgs.TGoAway).SetGoAway(ga)
	c.sendMsg(msg)
}

// handleOpen processes an Open message.
func (c *ServerConn) handleOpen(ctx context.Context, open msgs.Open) {
	descr := open.Descr()

	// Look up handler.
	handler, ok := c.server.registry.LookupByDescr(descr)
	if !ok {
		c.sendOpenAck(open.OpenID(), 0, msgs.ErrUnimplemented, "no handler registered")
		return
	}

	// Validate RPC type matches.
	if handler.Type() != descr.Type() {
		c.sendOpenAck(open.OpenID(), 0, msgs.ErrInvalidArgument, "RPC type mismatch")
		return
	}

	// Create session.
	c.mu.Lock()
	sessionID := c.nextSessionID
	c.nextSessionID++

	// Convert metadata list to slice.
	md := make([]msgs.Metadata, open.MetadataLen())
	for i := 0; i < open.MetadataLen(); i++ {
		md[i] = open.MetadataGet(i)
	}
	sess := newServerSession(sessionID, descr.Type(), handler, md)
	c.sessions[sessionID] = sess
	c.mu.Unlock()

	// Send OpenAck.
	c.sendOpenAck(open.OpenID(), sessionID, msgs.ErrNone, "")

	// Create context with deadline if specified.
	handlerCtx := ctx
	var cancel context.CancelFunc
	if open.DeadlineMS() > 0 {
		deadline := time.Now().Add(time.Duration(open.DeadlineMS()) * time.Millisecond)
		handlerCtx, cancel = context.WithDeadline(ctx, deadline)
	}

	// Start handler in pool.
	pool := context.Pool(ctx)
	pool.Submit(ctx, func() {
		if cancel != nil {
			defer cancel()
		}
		c.runHandler(handlerCtx, sess)
	})
}

// runHandler runs the appropriate handler for a session.
func (c *ServerConn) runHandler(ctx context.Context, sess *serverSession) {
	defer func() {
		c.mu.Lock()
		delete(c.sessions, sess.id)
		c.mu.Unlock()
		sess.close()
	}()

	var err error

	switch h := sess.handler.(type) {
	case SyncHandler:
		err = c.runSyncHandler(ctx, sess, h)
	case BiDirHandler:
		stream := newBiDirStream(sess.id, c, sess.recvCh, sess.cancelCh)
		err = h.HandleFunc(ctx, stream)
		stream.close()
		// Send EndStream to signal we're done sending.
		c.sendPayload(sess.id, 0, nil, true)
	case SendHandler:
		stream := newRecvStream(sess.id, c, sess.recvCh, sess.cancelCh)
		err = h.HandleFunc(ctx, stream)
	case RecvHandler:
		stream := newSendStream(sess.id, c, sess.cancelCh)
		err = h.HandleFunc(ctx, stream)
		stream.close()
		// Send EndStream to signal we're done.
		c.sendPayload(sess.id, 0, nil, true)
	default:
		err = fmt.Errorf("unknown handler type: %T", sess.handler)
	}

	// Send Close with error if any.
	errCode := errCodeFromError(err)
	errMsg := errorMessage(err)
	c.sendClose(sess.id, errCode, errMsg)
}

// runSyncHandler handles synchronous request/response sessions.
func (c *ServerConn) runSyncHandler(ctx context.Context, sess *serverSession, h SyncHandler) error {
	for p := range sess.recvCh {
		resp, err := h.HandleFunc(ctx, p.Payload(), sess.metadata)
		if err != nil {
			return err
		}
		if err := c.sendPayload(sess.id, p.ReqID(), resp, false); err != nil {
			return err
		}
	}
	return nil
}

// handleClose processes a Close message from the client.
func (c *ServerConn) handleClose(cl msgs.Close) {
	c.mu.Lock()
	sess, ok := c.sessions[cl.SessionID()]
	if ok {
		delete(c.sessions, cl.SessionID())
	}
	c.mu.Unlock()

	if ok {
		close(sess.recvCh)
	}
}

// handlePayload processes a Payload message.
func (c *ServerConn) handlePayload(p msgs.Payload) {
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

// handleCancel processes a Cancel message.
func (c *ServerConn) handleCancel(cancel msgs.Cancel) {
	c.mu.Lock()
	sess, ok := c.sessions[cancel.SessionID()]
	if ok {
		delete(c.sessions, cancel.SessionID())
	}
	c.mu.Unlock()

	if ok {
		sess.close()       // Close cancelCh so handlePayload doesn't block
		close(sess.recvCh) // Signal handler to drain and exit
	}
}

// handlePing processes a Ping message.
func (c *ServerConn) handlePing(ping msgs.Ping) {
	pong := msgs.NewPong(c.ctx).SetID(ping.ID())
	msg := msgs.NewMsg(c.ctx).SetType(msgs.TPong).SetPong(pong)
	c.sendMsg(msg)
}

// sendMsg sends a message on the transport.
func (c *ServerConn) sendMsg(msg msgs.Msg) error {
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

// sendOpenAck sends an OpenAck message.
func (c *ServerConn) sendOpenAck(openID, sessionID uint32, errCode msgs.ErrCode, errMsg string) error {
	ack := msgs.NewOpenAck(c.ctx).
		SetOpenID(openID).
		SetSessionID(sessionID).
		SetProtocolMajor(ProtocolMajor).
		SetProtocolMinor(ProtocolMinor).
		SetErrCode(errCode).
		SetError(errMsg)

	msg := msgs.NewMsg(c.ctx).SetType(msgs.TOpenAck).SetOpenAck(ack)
	return c.sendMsg(msg)
}

// sendClose sends a Close message.
func (c *ServerConn) sendClose(sessionID uint32, errCode msgs.ErrCode, errMsg string) error {
	cl := msgs.NewClose(c.ctx).
		SetSessionID(sessionID).
		SetErrCode(errCode).
		SetError(errMsg)

	msg := msgs.NewMsg(c.ctx).SetType(msgs.TClose).SetClose(cl)
	return c.sendMsg(msg)
}

// sendPayload sends a Payload message.
func (c *ServerConn) sendPayload(sessionID, reqID uint32, payload []byte, endStream bool) error {
	p := msgs.NewPayload(c.ctx).
		SetSessionID(sessionID).
		SetReqID(reqID).
		SetPayload(payload).
		SetEndStream(endStream)

	msg := msgs.NewMsg(c.ctx).SetType(msgs.TPayload).SetPayload(p)
	return c.sendMsg(msg)
}
