// Package client provides RPC client functionality for multiplexed connections.
package client

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"sync/atomic"
	"time"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/compress"
	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/metadata"
	"github.com/bearlytools/claw/rpc/retry"
)

// Protocol version constants.
const (
	ProtocolMajor = 1
	ProtocolMinor = 0
)

// Common errors.
var (
	ErrClosed             = errors.New("connection closed")
	ErrSessionClosed      = errors.New("session closed")
	ErrFatalError         = errors.New("fatal connection error")
	ErrTimeout            = errors.New("operation timed out")
	ErrCanceled           = errors.New("operation canceled")
	ErrMessageTooLarge    = errors.New("message size exceeds limit")
	ErrInsecureTransport  = errors.New("credentials require transport security but connection is not secure")
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

// WithCompression sets the default compression algorithm for outgoing payloads.
// Use msgs.CmpNone to disable compression (default).
func WithCompression(alg msgs.Compression) Option {
	return func(c *Conn) {
		c.defaultCompression = alg
	}
}

// WithMaxRecvMsgSize sets the maximum size of messages the client will accept.
// Messages larger than this are rejected. Default is 4MB (same as maxPayloadSize).
// Set to 0 to use the default.
func WithMaxRecvMsgSize(size int) Option {
	return func(c *Conn) {
		c.maxRecvMsgSize = size
	}
}

// WithMaxSendMsgSize sets the maximum size of messages the client will send.
// Attempts to send larger messages return ErrMessageTooLarge.
// Default is 0 (no limit beyond maxPayloadSize).
func WithMaxSendMsgSize(size int) Option {
	return func(c *Conn) {
		c.maxSendMsgSize = size
	}
}

// WithSecure marks the connection as using transport-level security (TLS).
// This is used by PerRPCCredentials that require transport security.
// Transports should call this option when establishing a TLS connection.
func WithSecure(secure bool) Option {
	return func(c *Conn) {
		c.secure = secure
	}
}

// WithUnaryInterceptor adds unary interceptors to the client connection.
// Multiple calls chain the interceptors; they execute in the order provided.
func WithUnaryInterceptor(interceptors ...interceptor.UnaryClientInterceptor) Option {
	return func(c *Conn) {
		if c.unaryInterceptor == nil {
			c.unaryInterceptor = interceptor.ChainUnaryClient(interceptors...)
		} else {
			c.unaryInterceptor = interceptor.ChainUnaryClient(append([]interceptor.UnaryClientInterceptor{c.unaryInterceptor}, interceptors...)...)
		}
	}
}

// WithStreamInterceptor adds stream interceptors to the client connection.
// Multiple calls chain the interceptors; they execute in the order provided.
func WithStreamInterceptor(interceptors ...interceptor.StreamClientInterceptor) Option {
	return func(c *Conn) {
		if c.streamInterceptor == nil {
			c.streamInterceptor = interceptor.ChainStreamClient(interceptors...)
		} else {
			c.streamInterceptor = interceptor.ChainStreamClient(append([]interceptor.StreamClientInterceptor{c.streamInterceptor}, interceptors...)...)
		}
	}
}

// WithRetryPolicy adds a retry interceptor with the given policy for unary calls.
// The retry interceptor is prepended to any existing unary interceptors.
func WithRetryPolicy(policy retry.Policy) Option {
	return func(c *Conn) {
		retryInterceptor := retry.UnaryClientInterceptor(policy)
		if c.unaryInterceptor == nil {
			c.unaryInterceptor = retryInterceptor
		} else {
			c.unaryInterceptor = interceptor.ChainUnaryClient(retryInterceptor, c.unaryInterceptor)
		}
	}
}

// CallOption configures a single RPC call.
type CallOption func(*callOptions)

// callOptions holds per-call configuration.
type callOptions struct {
	metadata     metadata.MD
	waitForReady bool
	creds        PerRPCCredentials
}

// PerRPCCredentials provides credentials for each RPC call.
// This is similar to gRPC's PerRPCCredentials interface.
type PerRPCCredentials interface {
	// GetRequestMetadata returns metadata to attach to each RPC.
	// The context allows credentials to be retrieved dynamically
	// (e.g., refreshing tokens). The uri is the target of the call
	// in "package/service/method" format.
	GetRequestMetadata(ctx context.Context, uri string) (map[string]string, error)

	// RequireTransportSecurity returns true if the credentials require
	// transport-level security (TLS). If true and the connection is not
	// secure, calls will fail with an error.
	RequireTransportSecurity() bool
}

// WithWaitForReady configures the call to block until the connection is ready
// or the context times out, rather than failing immediately if the connection
// is not ready. Default is false (fail fast).
func WithWaitForReady(wait bool) CallOption {
	return func(o *callOptions) {
		o.waitForReady = wait
	}
}

// WithMetadata sets metadata to send with the RPC call.
func WithMetadata(md metadata.MD) CallOption {
	return func(o *callOptions) {
		o.metadata = md
	}
}

// WithPerRPCCredentials attaches credentials to the RPC call.
// The credentials are fetched dynamically via GetRequestMetadata
// and merged into the call's metadata.
func WithPerRPCCredentials(creds PerRPCCredentials) CallOption {
	return func(o *callOptions) {
		o.creds = creds
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
	respMD    metadata.MD // Metadata from OpenAck
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
	readyCh  chan struct{} // Closed when connection is ready for use
	fatalErr error

	pingInterval       time.Duration
	pingTimeout        time.Duration
	maxPayloadSize     uint32
	maxRecvMsgSize     int // Maximum size of received messages (0 = default 4MB)
	maxSendMsgSize     int // Maximum size of sent messages (0 = no limit)
	defaultCompression msgs.Compression
	secure             bool // True if transport is secured (TLS)

	// Keepalive state (times stored as UnixNano)
	lastActivity atomic.Int64  // Last time we sent or received data
	pongCh       chan struct{} // Signals pong received

	unaryInterceptor  interceptor.UnaryClientInterceptor
	streamInterceptor interceptor.StreamClientInterceptor

	ctx context.Context
}

// New creates a new connection over the given transport.
func New(ctx context.Context, transport io.ReadWriteCloser, opts ...Option) *Conn {
	readyCh := make(chan struct{})
	close(readyCh) // Transport is already connected, so connection is immediately ready

	c := &Conn{
		transport:      transport,
		sessions:       make(map[uint32]*session),
		pending:        make(map[uint32]*session),
		closed:         make(chan struct{}),
		readyCh:        readyCh,
		pingInterval:   30 * time.Second,
		pingTimeout:    10 * time.Second,
		maxPayloadSize: 4 * 1024 * 1024, // 4MB default
		pongCh:         make(chan struct{}, 1),
		ctx:            ctx,
	}

	// Initialize lastActivity to now
	c.lastActivity.Store(time.Now().UnixNano())

	for _, opt := range opts {
		opt(c)
	}

	// Start the read loop and ping loop in the pool.
	pool := context.Pool(ctx)
	pool.Submit(ctx, func() {
		c.readLoop()
	})
	pool.Submit(ctx, func() {
		c.pingLoop()
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

// Ready returns a channel that is closed when the connection is ready for use.
// This can be used with select to wait for the connection to become ready.
func (c *Conn) Ready() <-chan struct{} {
	return c.readyCh
}

// IsReady returns true if the connection is ready for use.
// A connection is ready if it's not closed and has no fatal error.
func (c *Conn) IsReady() bool {
	select {
	case <-c.closed:
		return false
	default:
		c.mu.Lock()
		ready := c.fatalErr == nil
		c.mu.Unlock()
		return ready
	}
}

// waitForReady blocks until the connection is ready or context is done.
// If waitForReady is false, it checks readiness immediately and returns an error if not ready.
func (c *Conn) waitForReady(ctx context.Context, waitForReady bool) error {
	// Fast path: check if already closed
	select {
	case <-c.closed:
		if err := c.Err(); err != nil {
			return err
		}
		return ErrClosed
	default:
	}

	// If not waiting, just check current state
	if !waitForReady {
		if !c.IsReady() {
			if err := c.Err(); err != nil {
				return err
			}
			return ErrClosed
		}
		return nil
	}

	// Wait for ready or context cancellation
	select {
	case <-c.readyCh:
		// Check if connection is actually usable
		if !c.IsReady() {
			if err := c.Err(); err != nil {
				return err
			}
			return ErrClosed
		}
		return nil
	case <-c.closed:
		if err := c.Err(); err != nil {
			return err
		}
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
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

// markActivity updates the last activity time for keepalive tracking.
func (c *Conn) markActivity() {
	c.lastActivity.Store(time.Now().UnixNano())
}

// readLoop reads messages from the transport and dispatches them.
func (c *Conn) readLoop() {
	// Close all recvCh when readLoop exits. This signals receivers to drain and exit.
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
			return
		case <-c.ctx.Done():
			return
		default:
		}

		msg := msgs.NewMsg(c.ctx)
		_, err := msg.UnmarshalReader(c.transport)
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.setFatalError(io.EOF)
				return
			}
			c.setFatalError(fmt.Errorf("read error: %w", err))
			return
		}

		// Mark activity for keepalive tracking
		c.markActivity()

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

	// Extract metadata from OpenAck.
	mdLen := ack.MetadataLen(c.ctx)
	if mdLen > 0 {
		mds := make([]msgs.Metadata, mdLen)
		for i := 0; i < mdLen; i++ {
			mds[i] = ack.MetadataGet(c.ctx, i)
		}
		sess.respMD = metadata.FromMsgs(c.ctx, mds)
	}

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

	// Send Close to closeCh for error info, then close recvCh to signal end of stream.
	select {
	case sess.closeCh <- cl:
	default:
	}
	close(sess.recvCh)
}

// handlePayload processes a Payload message.
func (c *Conn) handlePayload(p msgs.Payload) {
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
	maxSize := c.maxRecvMsgSize
	if maxSize == 0 {
		maxSize = int(c.maxPayloadSize) // Use default
	}
	if maxSize > 0 && len(p.Payload()) > maxSize {
		// Message too large, drop it.
		return
	}

	select {
	case sess.recvCh <- p:
	case <-sess.cancelCh:
	}
}

// handlePong processes a Pong message.
func (c *Conn) handlePong(p msgs.Pong) {
	// Signal the ping loop that we received a pong.
	// Non-blocking send in case ping loop isn't waiting.
	select {
	case c.pongCh <- struct{}{}:
	default:
	}
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
func (c *Conn) openSession(ctx context.Context, pkg, service, call string, rpcType msgs.RPCType, md metadata.MD) (*session, error) {
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
	descr := msgs.NewDescr(ctx).
		SetPackage(pkg).
		SetService(service).
		SetCall(call).
		SetType(rpcType)

	open := msgs.NewOpen(ctx).
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

	// Add metadata if provided.
	if md != nil {
		mds := md.ToMsgs(ctx)
		open.MetadataAppend(ctx, mds...)
	}

	msg := msgs.NewMsg(ctx).SetType(msgs.TOpen).SetOpen(open)

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
	cl := msgs.NewClose(c.ctx).
		SetSessionID(sessionID).
		SetErrCode(errCode).
		SetError(errMsg)

	msg := msgs.NewMsg(c.ctx).SetType(msgs.TClose).SetClose(cl)
	return c.sendMsg(msg)
}

// sendPayload sends a Payload message.
func (c *Conn) sendPayload(sessionID, reqID uint32, payload []byte, endStream bool) error {
	// Check send size limit before compression (check uncompressed size).
	if c.maxSendMsgSize > 0 && len(payload) > c.maxSendMsgSize {
		return fmt.Errorf("%w: %d bytes exceeds send limit of %d", ErrMessageTooLarge, len(payload), c.maxSendMsgSize)
	}

	// Compress if compression is configured and there's data to compress.
	compressed := payload
	compression := c.defaultCompression
	if compression != msgs.CmpNone && len(payload) > 0 {
		var err error
		compressed, err = compress.Compress(compression, payload)
		if err != nil {
			return fmt.Errorf("compression failed: %w", err)
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

// sendCancel sends a Cancel message.
func (c *Conn) sendCancel(sessionID, reqID uint32) error {
	cancel := msgs.NewCancel(c.ctx).
		SetSessionID(sessionID).
		SetReqID(reqID)

	msg := msgs.NewMsg(c.ctx).SetType(msgs.TCancel).SetCancel(cancel)
	return c.sendMsg(msg)
}

// applyCredentials applies per-RPC credentials to metadata.
// Returns the updated metadata and any error from credential retrieval.
func (c *Conn) applyCredentials(ctx context.Context, opts *callOptions, uri string) (metadata.MD, error) {
	md := opts.metadata
	if md == nil {
		md, _ = metadata.FromContext(ctx)
	}

	if opts.creds == nil {
		return md, nil
	}

	// Check transport security requirement.
	if opts.creds.RequireTransportSecurity() && !c.secure {
		return nil, ErrInsecureTransport
	}

	// Get credential metadata.
	credMD, err := opts.creds.GetRequestMetadata(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	// Merge credential metadata.
	if len(credMD) > 0 {
		if md == nil {
			md = metadata.MD{}
		}
		for k, v := range credMD {
			md.SetString(k, v)
		}
	}

	return md, nil
}

// Sync creates a new synchronous RPC client.
func (c *Conn) Sync(ctx context.Context, pkg, service, call string, opts ...CallOption) (*SyncClient, error) {
	var callOpts callOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// Wait for connection to be ready if requested.
	if err := c.waitForReady(ctx, callOpts.waitForReady); err != nil {
		return nil, err
	}

	// Apply per-RPC credentials.
	uri := pkg + "/" + service + "/" + call
	md, err := c.applyCredentials(ctx, &callOpts, uri)
	if err != nil {
		return nil, err
	}

	sess, err := c.openSession(ctx, pkg, service, call, msgs.RTSynchronous, md)
	if err != nil {
		return nil, err
	}

	return &SyncClient{
		conn:      c,
		sessionID: sess.id,
		session:   sess,
		pending:   make(map[uint32]chan response),
		method:    uri,
	}, nil
}

// BiDir creates a new bidirectional streaming RPC client.
func (c *Conn) BiDir(ctx context.Context, pkg, service, call string, opts ...CallOption) (*BiDirClient, error) {
	var callOpts callOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// Wait for connection to be ready if requested.
	if err := c.waitForReady(ctx, callOpts.waitForReady); err != nil {
		return nil, err
	}

	// Apply per-RPC credentials.
	uri := pkg + "/" + service + "/" + call
	md, err := c.applyCredentials(ctx, &callOpts, uri)
	if err != nil {
		return nil, err
	}

	sess, err := c.openSession(ctx, pkg, service, call, msgs.RTBiDirectional, md)
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
func (c *Conn) Send(ctx context.Context, pkg, service, call string, opts ...CallOption) (*SendClient, error) {
	var callOpts callOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// Wait for connection to be ready if requested.
	if err := c.waitForReady(ctx, callOpts.waitForReady); err != nil {
		return nil, err
	}

	// Apply per-RPC credentials.
	uri := pkg + "/" + service + "/" + call
	md, err := c.applyCredentials(ctx, &callOpts, uri)
	if err != nil {
		return nil, err
	}

	sess, err := c.openSession(ctx, pkg, service, call, msgs.RTSend, md)
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
func (c *Conn) Recv(ctx context.Context, pkg, service, call string, opts ...CallOption) (*RecvClient, error) {
	var callOpts callOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// Wait for connection to be ready if requested.
	if err := c.waitForReady(ctx, callOpts.waitForReady); err != nil {
		return nil, err
	}

	// Apply per-RPC credentials.
	uri := pkg + "/" + service + "/" + call
	md, err := c.applyCredentials(ctx, &callOpts, uri)
	if err != nil {
		return nil, err
	}

	sess, err := c.openSession(ctx, pkg, service, call, msgs.RTRecv, md)
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
// When context is cancelled, it drains and yields any buffered messages before returning.
func recvIter(ctx context.Context, sess *session, errPtr *error) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for {
			select {
			case <-ctx.Done():
				// Context cancelled - drain and yield buffered messages before returning.
				for {
					select {
					case p, ok := <-sess.recvCh:
						if !ok {
							goto checkClose
						}
						payload := p.Payload()
						if p.EndStream() && len(payload) == 0 {
							goto checkClose
						}
						if !yield(payload) {
							return
						}
						if p.EndStream() {
							goto checkClose
						}
					default:
						// No more buffered messages.
						goto checkClose
					}
				}
			case p, ok := <-sess.recvCh:
				if !ok {
					goto checkClose
				}
				payload := p.Payload()
				if p.EndStream() && len(payload) == 0 {
					goto checkClose
				}
				if !yield(payload) {
					return
				}
				if p.EndStream() {
					goto checkClose
				}
			}
		}

	checkClose:
		// Check for error info from Close message.
		select {
		case cl := <-sess.closeCh:
			if cl.ErrCode() != msgs.ErrNone {
				*errPtr = fmt.Errorf("session closed with error: %s", cl.Error())
			}
		default:
		}
	}
}

// pingLoop periodically sends keepalive pings when the connection is idle.
func (c *Conn) pingLoop() {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.closed:
			return
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Check if we've been idle for pingInterval
			lastAct := time.Unix(0, c.lastActivity.Load())
			if time.Since(lastAct) < c.pingInterval {
				// Not idle, skip this ping
				continue
			}

			// Drain any stale pong signals
			select {
			case <-c.pongCh:
			default:
			}

			// Send ping
			ping := msgs.NewPing(c.ctx)
			msg := msgs.NewMsg(c.ctx).SetType(msgs.TPing).SetPing(ping)
			if err := c.sendMsg(msg); err != nil {
				// sendMsg already calls setFatalError
				return
			}

			// Wait for pong with timeout
			select {
			case <-c.closed:
				return
			case <-c.ctx.Done():
				return
			case <-c.pongCh:
				// Got pong, connection is healthy
			case <-time.After(c.pingTimeout):
				// No pong received, connection is dead
				c.setFatalError(fmt.Errorf("keepalive timeout: no pong received within %v", c.pingTimeout))
				return
			}
		}
	}
}

// biDirClientAdapter wraps BiDirClient to implement interceptor.ClientStream.
type biDirClientAdapter struct {
	*BiDirClient
}

func (a *biDirClientAdapter) Send(ctx context.Context, payload []byte) error {
	return a.BiDirClient.Send(ctx, payload)
}

func (a *biDirClientAdapter) Recv(ctx context.Context) iter.Seq[[]byte] {
	return a.BiDirClient.Recv(ctx)
}

// Compile-time check that biDirClientAdapter implements interceptor.ClientStream.
var _ interceptor.ClientStream = (*biDirClientAdapter)(nil)

// sendClientAdapter wraps SendClient to implement interceptor.ClientStream.
type sendClientAdapter struct {
	*SendClient
}

func (a *sendClientAdapter) Send(ctx context.Context, payload []byte) error {
	return a.SendClient.Send(ctx, payload)
}

func (a *sendClientAdapter) Recv(ctx context.Context) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {}
}

func (a *sendClientAdapter) CloseSend() error {
	return nil // Close() handles this for SendClient
}

func (a *sendClientAdapter) Err() error {
	return nil
}

// Compile-time check that sendClientAdapter implements interceptor.ClientStream.
var _ interceptor.ClientStream = (*sendClientAdapter)(nil)

// recvClientAdapter wraps RecvClient to implement interceptor.ClientStream.
type recvClientAdapter struct {
	*RecvClient
}

func (a *recvClientAdapter) Send(ctx context.Context, payload []byte) error {
	return nil
}

func (a *recvClientAdapter) Recv(ctx context.Context) iter.Seq[[]byte] {
	return a.RecvClient.Recv(ctx)
}

func (a *recvClientAdapter) CloseSend() error {
	return nil
}

// Compile-time check that recvClientAdapter implements interceptor.ClientStream.
var _ interceptor.ClientStream = (*recvClientAdapter)(nil)
