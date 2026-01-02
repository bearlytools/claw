package server

import (
	"fmt"
	"io"
	stdsync "sync"
	"time"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/concurrency/worker"
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/values/sizes"

	"github.com/bearlytools/claw/languages/go/pack"
	"github.com/bearlytools/claw/rpc/compress"
	"github.com/bearlytools/claw/rpc/errors"
	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/metadata"
)

// Protocol version constants.
const (
	ProtocolMajor = 1
	ProtocolMinor = 0
)

// Default max payload size (4 MiB).
const defaultMaxPayloadSize = 4 * sizes.MiB

// serverSession represents a server-side session.
type serverSession struct {
	id        uint32
	rpcType   msgs.RPCType
	handler   Handler
	recvCh    chan msgs.Payload
	cancelCh  chan struct{}
	closeOnce sync.Once
	metadata  []msgs.Metadata

	pkg     string
	service string
	method  string
}

func newServerSession(id uint32, rpcType msgs.RPCType, handler Handler, md []msgs.Metadata, pkg, service, method string) *serverSession {
	return &serverSession{
		id:       id,
		rpcType:  rpcType,
		handler:  handler,
		recvCh:   make(chan msgs.Payload, 16),
		cancelCh: make(chan struct{}),
		metadata: md,
		pkg:      pkg,
		service:  service,
		method:   method,
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

	nextSessionID      uint32
	defaultCompression msgs.Compression

	maxRecvMsgSize int // Maximum size of received messages (from Server)
	maxSendMsgSize int // Maximum size of sent messages (from Server)

	pool *worker.Pool // Pool for RPC handlers (nil = use context.Pool)

	ctx context.Context

	// Graceful shutdown state
	draining       bool              // True when draining (no new sessions allowed)
	activeHandlers stdsync.WaitGroup // Tracks in-flight handlers
	drained        chan struct{}     // Closed when all handlers complete during draining

	// Packing state
	allowPacking   bool // Server allows packing (from Server config)
	packingEnabled bool // Whether packing is enabled (negotiated with client)
}

func newServerConn(ctx context.Context, server *Server, transport io.ReadWriteCloser, compression msgs.Compression) *ServerConn {
	c := &ServerConn{
		server:             server,
		transport:          transport,
		sessions:           make(map[uint32]*serverSession),
		closed:             make(chan struct{}),
		nextSessionID:      1,
		defaultCompression: compression,
		maxRecvMsgSize:     server.maxRecvMsgSize,
		maxSendMsgSize:     server.maxSendMsgSize,
		ctx:                ctx,
		allowPacking:       server.allowPacking,
	}

	// Create a limited pool if maxConcurrentRPCs is set.
	if server.maxConcurrentRPCs > 0 {
		c.pool = context.Pool(ctx).Limited(ctx, "claw-rpc-handlers", server.maxConcurrentRPCs)
	}

	return c
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
		var err error

		// Read packed or unpacked message.
		// Open messages are never packed. After first OpenAck with packing=true,
		// all subsequent messages from client are packed.
		if c.packingEnabled {
			err = c.readPackedMsg(msg)
		} else {
			_, err = msg.UnmarshalReader(c.transport)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.setFatalError(io.EOF)
				return nil
			}
			c.setFatalError(errors.E(c.ctx, errors.Internal, fmt.Errorf("read error: %w", err)))
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

// readPackedMsg reads a packed message from the transport and unmarshals it.
func (c *ServerConn) readPackedMsg(msg msgs.Msg) error {
	// Read pack header (16 bytes: 8 unpacked size + 8 packed size).
	var header [pack.HeaderSize]byte
	if _, err := io.ReadFull(c.transport, header[:]); err != nil {
		return err
	}

	packedSize := pack.PackedSize(header[:])

	// Read packed data (header + packed body).
	packedData := make([]byte, pack.HeaderSize+packedSize)
	copy(packedData, header[:])
	if _, err := io.ReadFull(c.transport, packedData[pack.HeaderSize:]); err != nil {
		return err
	}

	// Unpack data.
	unpacked, err := pack.Unpack(c.ctx, packedData)
	if err != nil {
		return errors.E(c.ctx, errors.Internal, fmt.Errorf("unpack error: %w", err))
	}
	defer unpacked.Release(c.ctx)

	// Unmarshal into message.
	return msg.Unmarshal(unpacked.Bytes())
}

// Close closes the connection immediately without waiting for handlers.
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

// GracefulClose gracefully closes the connection, waiting for in-flight
// handlers to complete. The context controls how long to wait; if it's
// cancelled or times out, remaining handlers are forcefully terminated.
//
// Returns nil if all handlers completed gracefully, or an error if the
// context was cancelled before all handlers finished.
func (c *ServerConn) GracefulClose(ctx context.Context) error {
	c.mu.Lock()

	// Check if already closed.
	select {
	case <-c.closed:
		c.mu.Unlock()
		return nil
	default:
	}

	// Enter draining mode - no new sessions will be accepted.
	c.draining = true

	// Send GoAway to client.
	c.mu.Unlock()
	c.goAway(ctx)

	// Wait for all active handlers to complete.
	// Note: We don't monitor c.closed here because if the connection closes
	// while we're waiting (e.g., client disconnects after receiving GoAway),
	// the serve() loop will close all recvCh channels, causing handlers to
	// exit and eventually call Done() on the WaitGroup.
	done := make(chan struct{})
	go func() {
		c.activeHandlers.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All handlers completed gracefully.
		return c.Close()
	case <-ctx.Done():
		// Timeout - force close remaining handlers.
		c.Close()
		return ctx.Err()
	}
}

// IsDraining returns true if the connection is draining (no new sessions allowed).
func (c *ServerConn) IsDraining() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.draining
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

	// Check if draining - reject new sessions.
	c.mu.Lock()
	if c.draining {
		c.mu.Unlock()
		c.sendOpenAck(open.OpenID(), 0, msgs.ErrUnavailable, "server is draining", false)
		return
	}
	c.mu.Unlock()

	// Look up handler.
	handler, ok := c.server.registry.LookupByDescr(descr)
	if !ok {
		c.sendOpenAck(open.OpenID(), 0, msgs.ErrUnimplemented, "no handler registered", false)
		return
	}

	// Validate RPC type matches.
	if handler.Type() != descr.Type() {
		c.sendOpenAck(open.OpenID(), 0, msgs.ErrInvalidArgument, "RPC type mismatch", false)
		return
	}

	// Create session.
	c.mu.Lock()
	// Double-check draining after acquiring lock.
	if c.draining {
		c.mu.Unlock()
		c.sendOpenAck(open.OpenID(), 0, msgs.ErrUnavailable, "server is draining", false)
		return
	}

	sessionID := c.nextSessionID
	c.nextSessionID++

	// Negotiate packing on first Open.
	// If client requests packing and server allows it, enable packing.
	if sessionID == 1 && open.Packing() && c.allowPacking {
		c.packingEnabled = true
	}

	// Convert metadata list to slice.
	md := make([]msgs.Metadata, open.MetadataLen(ctx))
	for i := 0; i < open.MetadataLen(ctx); i++ {
		md[i] = open.MetadataGet(ctx, i)
	}
	sess := newServerSession(sessionID, descr.Type(), handler, md, descr.Package(), descr.Service(), descr.Call())
	c.sessions[sessionID] = sess

	// Track this handler in the WaitGroup before releasing the lock.
	c.activeHandlers.Add(1)
	c.mu.Unlock()

	// Send OpenAck with packing status.
	c.sendOpenAck(open.OpenID(), sessionID, msgs.ErrNone, "", c.packingEnabled)

	// Create context with deadline if specified.
	handlerCtx := ctx
	var cancel context.CancelFunc
	if open.DeadlineMS() > 0 {
		deadline := time.Now().Add(time.Duration(open.DeadlineMS()) * time.Millisecond)
		handlerCtx, cancel = context.WithDeadline(ctx, deadline)
	}

	// Start handler in pool.
	pool := c.pool
	if pool == nil {
		pool = context.Pool(ctx)
	}
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

		// Signal that this handler is done.
		c.activeHandlers.Done()
	}()

	var err error
	var trailer metadata.MD

	switch h := sess.handler.(type) {
	case SyncHandler:
		err = c.runSyncHandler(ctx, sess, h)
	case BiDirHandler:
		stream := newBiDirStream(ctx, sess.id, c, sess.recvCh, sess.cancelCh)
		err = c.runBiDirHandler(ctx, sess, stream, h)
		trailer = stream.Trailer()
		stream.close()
		// Send EndStream to signal we're done sending.
		c.sendPayload(sess.id, 0, nil, true)
	case SendHandler:
		stream := newRecvStream(ctx, sess.id, c, sess.recvCh, sess.cancelCh)
		err = c.runSendHandler(ctx, sess, stream, h)
		trailer = stream.Trailer()
	case RecvHandler:
		stream := newSendStream(ctx, sess.id, c, sess.cancelCh)
		err = c.runRecvHandler(ctx, sess, stream, h)
		trailer = stream.Trailer()
		stream.close()
		// Send EndStream to signal we're done.
		c.sendPayload(sess.id, 0, nil, true)
	default:
		err = errors.E(ctx, errors.Internal, fmt.Errorf("unknown handler type: %T", sess.handler))
	}

	// Send Close with error and trailer metadata if any.
	errCode := errCodeFromError(err)
	errMsg := errorMessage(err)
	c.sendClose(sess.id, errCode, errMsg, trailer)
}

// runBiDirHandler runs a bidirectional stream handler with interceptor support.
func (c *ServerConn) runBiDirHandler(ctx context.Context, sess *serverSession, stream *BiDirStream, h BiDirHandler) error {
	if c.server.streamInterceptor == nil {
		return h.HandleFunc(ctx, stream)
	}

	info := &interceptor.StreamServerInfo{
		Package:   sess.pkg,
		Service:   sess.service,
		Method:    sess.method,
		SessionID: sess.id,
		Metadata:  sess.metadata,
		RPCType:   sess.rpcType,
	}

	handler := func(ctx context.Context, s interceptor.ServerStream) error {
		return h.HandleFunc(ctx, s.(*BiDirStream))
	}

	return c.server.streamInterceptor(ctx, stream, info, handler)
}

// runSendHandler runs a send handler (client sends, server receives) with interceptor support.
func (c *ServerConn) runSendHandler(ctx context.Context, sess *serverSession, stream *RecvStream, h SendHandler) error {
	if c.server.streamInterceptor == nil {
		return h.HandleFunc(ctx, stream)
	}

	info := &interceptor.StreamServerInfo{
		Package:   sess.pkg,
		Service:   sess.service,
		Method:    sess.method,
		SessionID: sess.id,
		Metadata:  sess.metadata,
		RPCType:   sess.rpcType,
	}

	adapter := &recvStreamAdapter{stream: stream}
	handler := func(ctx context.Context, s interceptor.ServerStream) error {
		return h.HandleFunc(ctx, s.(*recvStreamAdapter).stream)
	}

	return c.server.streamInterceptor(ctx, adapter, info, handler)
}

// runRecvHandler runs a recv handler (server sends, client receives) with interceptor support.
func (c *ServerConn) runRecvHandler(ctx context.Context, sess *serverSession, stream *SendStream, h RecvHandler) error {
	if c.server.streamInterceptor == nil {
		return h.HandleFunc(ctx, stream)
	}

	info := &interceptor.StreamServerInfo{
		Package:   sess.pkg,
		Service:   sess.service,
		Method:    sess.method,
		SessionID: sess.id,
		Metadata:  sess.metadata,
		RPCType:   sess.rpcType,
	}

	adapter := &sendStreamAdapter{stream: stream}
	handler := func(ctx context.Context, s interceptor.ServerStream) error {
		return h.HandleFunc(ctx, s.(*sendStreamAdapter).stream)
	}

	return c.server.streamInterceptor(ctx, adapter, info, handler)
}

// runSyncHandler handles synchronous request/response sessions.
func (c *ServerConn) runSyncHandler(ctx context.Context, sess *serverSession, h SyncHandler) error {
	info := &interceptor.UnaryServerInfo{
		Package:   sess.pkg,
		Service:   sess.service,
		Method:    sess.method,
		SessionID: sess.id,
		Metadata:  sess.metadata,
	}

	handler := func(ctx context.Context, req []byte) ([]byte, error) {
		return h.HandleFunc(ctx, req, sess.metadata)
	}

	for p := range sess.recvCh {
		var resp []byte
		var err error

		if c.server.unaryInterceptor != nil {
			resp, err = c.server.unaryInterceptor(ctx, p.Payload(), info, handler)
		} else {
			resp, err = handler(ctx, p.Payload())
		}
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

	// Decompress payload if needed.
	if p.Compression() != msgs.CmpNone && len(p.Payload()) > 0 {
		decompressed, err := compress.Decompress(p.Compression(), p.Payload())
		if err != nil {
			// Log error and drop payload - can't recover from decompression failure.
			return
		}
		// Create new payload with decompressed data.
		p = msgs.NewPayload(c.ctx).
			SetSessionID(p.SessionID()).
			SetReqID(p.ReqID()).
			SetPayload(decompressed).
			SetEndStream(p.EndStream()).
			SetCompression(msgs.CmpNone)
	}

	// Check message size limit (after decompression).
	if c.maxRecvMsgSize > 0 && len(p.Payload()) > c.maxRecvMsgSize {
		// Message too large, drop it.
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
		return errors.E(c.ctx, errors.Unavailable, ErrClosed)
	default:
	}

	// Pack message if packing is enabled (except Open and OpenAck which are never packed).
	if c.packingEnabled && msg.Type() != msgs.TOpen && msg.Type() != msgs.TOpenAck {
		data, err := msg.Marshal()
		if err != nil {
			c.setFatalError(errors.E(c.ctx, errors.Internal, fmt.Errorf("marshal error: %w", err)))
			return err
		}

		packed, err := pack.Pack(c.ctx, data)
		if err != nil {
			c.setFatalError(errors.E(c.ctx, errors.Internal, fmt.Errorf("pack error: %w", err)))
			return err
		}
		defer packed.Release(c.ctx)

		_, err = c.transport.Write(packed.Bytes())
		if err != nil {
			c.setFatalError(errors.E(c.ctx, errors.Unavailable, fmt.Errorf("write error: %w", err)))
			return err
		}
		return nil
	}

	_, err := msg.MarshalWriter(c.transport)
	if err != nil {
		c.setFatalError(errors.E(c.ctx, errors.Unavailable, fmt.Errorf("write error: %w", err)))
		return err
	}
	return nil
}

// sendOpenAck sends an OpenAck message.
func (c *ServerConn) sendOpenAck(openID, sessionID uint32, errCode msgs.ErrCode, errMsg string, packing bool) error {
	ack := msgs.NewOpenAck(c.ctx).
		SetOpenID(openID).
		SetSessionID(sessionID).
		SetProtocolMajor(ProtocolMajor).
		SetProtocolMinor(ProtocolMinor).
		SetErrCode(errCode).
		SetError(errMsg).
		SetPacking(packing)

	msg := msgs.NewMsg(c.ctx).SetType(msgs.TOpenAck).SetOpenAck(ack)
	return c.sendMsg(msg)
}

// sendClose sends a Close message.
func (c *ServerConn) sendClose(sessionID uint32, errCode msgs.ErrCode, errMsg string, md metadata.MD) error {
	cl := msgs.NewClose(c.ctx).
		SetSessionID(sessionID).
		SetErrCode(errCode).
		SetError(errMsg)

	// Add trailer metadata if provided.
	if md != nil {
		mds := md.ToMsgs(c.ctx)
		cl.MetadataAppend(c.ctx, mds...)
	}

	msg := msgs.NewMsg(c.ctx).SetType(msgs.TClose).SetClose(cl)
	return c.sendMsg(msg)
}

// sendPayload sends a Payload message.
func (c *ServerConn) sendPayload(sessionID, reqID uint32, payload []byte, endStream bool) error {
	// Check message size limit (before compression, check uncompressed size).
	if c.maxSendMsgSize > 0 && len(payload) > c.maxSendMsgSize {
		return errors.E(c.ctx, errors.ResourceExhausted, ErrMessageTooLarge)
	}

	// Compress if compression is configured and there's data to compress.
	compressed := payload
	compression := c.defaultCompression
	if compression != msgs.CmpNone && len(payload) > 0 {
		var err error
		compressed, err = compress.Compress(compression, payload)
		if err != nil {
			return errors.E(c.ctx, errors.Internal, fmt.Errorf("compression failed: %w", err))
		}
	}

	p := msgs.NewPayload(c.ctx).
		SetSessionID(sessionID).
		SetReqID(reqID).
		SetPayload(compressed).
		SetEndStream(endStream).
		SetCompression(compression)

	msg := msgs.NewMsg(c.ctx).SetType(msgs.TPayload).SetPayload(p)
	return c.sendMsg(msg)
}
