// Package client provides RPC client functionality for multiplexed connections.
package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"iter"
	"sync/atomic"
	"time"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/values/sizes"

	"github.com/bearlytools/claw/languages/go/pack"
	"github.com/bearlytools/claw/rpc/compress"
	"github.com/bearlytools/claw/rpc/errors"
	"github.com/bearlytools/claw/rpc/hedge"
	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/metadata"
	"github.com/bearlytools/claw/rpc/retry"
	"github.com/bearlytools/claw/rpc/serviceconfig"
)

// Protocol version constants.
const (
	ProtocolMajor = 1
	ProtocolMinor = 0
)

// Common errors.
var (
	ErrClosed             = errors.New("connection closed")
	ErrDraining           = errors.New("connection is draining")
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
// Messages larger than this are rejected. Default is 4 MiB.
func WithMaxRecvMsgSize(size int) Option {
	return func(c *Conn) {
		c.maxRecvMsgSize = size
	}
}

// WithMaxSendMsgSize sets the maximum size of messages the client will send.
// Attempts to send larger messages return ErrMessageTooLarge. Default is 4 MiB.
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

// WithPacking enables Cap'n Proto-style message packing for this connection.
// When enabled, the client will request packing in the first Open message.
// If the server agrees, all subsequent messages (except Open/OpenAck) will be packed.
// Packing can significantly reduce message size by eliminating zero bytes.
func WithPacking(enabled bool) Option {
	return func(c *Conn) {
		c.usePacking = enabled
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

// WithHedgePolicy adds a hedge interceptor with the given policy for unary calls.
// Hedging sends the same request to multiple backends in parallel and uses the
// first response. This reduces tail latency but increases backend load.
//
// Hedging is disabled by default. Set MaxHedgedRequests > 0 to enable.
// The hedge interceptor is prepended to any existing unary interceptors,
// so hedging happens before retry.
//
// Note: Only use hedging for idempotent operations.
func WithHedgePolicy(policy hedge.Policy) Option {
	return func(c *Conn) {
		if policy.MaxHedgedRequests <= 0 {
			return // Disabled
		}
		hedgeInterceptor := hedge.UnaryClientInterceptor(policy)
		if c.unaryInterceptor == nil {
			c.unaryInterceptor = hedgeInterceptor
		} else {
			c.unaryInterceptor = interceptor.ChainUnaryClient(hedgeInterceptor, c.unaryInterceptor)
		}
	}
}

// WithServiceConfig sets the service configuration for per-method settings.
// The config allows setting default timeouts and wait-for-ready behavior
// on a per-method, per-service, or per-package basis.
//
// Timeouts from service config are only applied if the context does not
// already have a deadline. Per-call options override service config settings.
//
// Example:
//
//	cfg := serviceconfig.NewBuilder().
//	    WithDefaultTimeout(30 * time.Second).
//	    WithTimeout("myapp/UserService/*", 10 * time.Second).
//	    WithTimeout("myapp/UserService/SlowMethod", 60 * time.Second).
//	    Build()
//	conn := client.New(ctx, transport, client.WithServiceConfig(cfg))
func WithServiceConfig(cfg *serviceconfig.Config) Option {
	return func(c *Conn) {
		c.serviceConfig = cfg
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

// GetWaitForReady extracts the wait-for-ready setting from call options.
// This is useful for connection pools that need to check this before
// selecting a connection.
func GetWaitForReady(opts ...CallOption) bool {
	var co callOptions
	for _, opt := range opts {
		opt(&co)
	}
	return co.waitForReady
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
	draining bool          // True when draining (no new sessions allowed)
	drained  chan struct{} // Closed when all sessions are done
	readyCh  chan struct{} // Closed when connection is ready for use
	fatalErr error

	pingInterval       time.Duration
	pingTimeout        time.Duration
	maxPayloadSize     uint32
	maxRecvMsgSize     int // Maximum size of received messages (default 4 MiB)
	maxSendMsgSize     int // Maximum size of sent messages (default 4 MiB)
	defaultCompression msgs.Compression
	secure             bool // True if transport is secured (TLS)

	// Packing state
	usePacking        bool // Client wants to use packing (from option)
	packingNegotiated bool // Whether packing has been negotiated
	packingEnabled    bool // Whether packing is enabled (after negotiation)

	// Keepalive state (times stored as UnixNano)
	lastActivity atomic.Int64  // Last time we sent or received data
	pongCh       chan struct{} // Signals pong received

	unaryInterceptor  interceptor.UnaryClientInterceptor
	streamInterceptor interceptor.StreamClientInterceptor

	serviceConfig *serviceconfig.Config // Per-method configuration

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
		maxPayloadSize: 4 * sizes.MiB,
		maxRecvMsgSize: 4 * sizes.MiB,
		maxSendMsgSize: 4 * sizes.MiB,
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

// Close closes the connection and all sessions immediately.
// For graceful shutdown that waits for in-flight RPCs, use GracefulClose.
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

// GracefulClose stops accepting new RPCs and waits for in-flight RPCs to complete
// before closing the connection. The context controls how long to wait; if the
// context is cancelled or times out, remaining sessions are forcefully closed.
//
// Returns nil if all sessions completed gracefully, or an error if the context
// was cancelled before all sessions finished.
func (c *Conn) GracefulClose(ctx context.Context) error {
	c.mu.Lock()

	// Check if already closed
	select {
	case <-c.closed:
		c.mu.Unlock()
		return nil
	default:
	}

	// Enter draining mode - no new sessions allowed
	c.draining = true

	// If no sessions, we're done
	if len(c.sessions) == 0 && len(c.pending) == 0 {
		c.mu.Unlock()
		return c.Close()
	}

	// Create drained channel if not exists
	if c.drained == nil {
		c.drained = make(chan struct{})
	}
	drained := c.drained
	c.mu.Unlock()

	// Wait for all sessions to complete or context to timeout
	select {
	case <-drained:
		// All sessions completed gracefully
		return c.Close()
	case <-ctx.Done():
		// Timeout - force close remaining sessions
		c.Close()
		return ctx.Err()
	case <-c.closed:
		// Connection was closed by other means
		return nil
	}
}

// IsDraining returns true if the connection is draining (no new sessions allowed).
func (c *Conn) IsDraining() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.draining
}

// checkDrained checks if draining is complete and signals the drained channel.
// Must be called when a session is removed.
func (c *Conn) checkDrained() {
	if !c.draining {
		return
	}
	if len(c.sessions) == 0 && len(c.pending) == 0 && c.drained != nil {
		select {
		case <-c.drained:
			// Already closed
		default:
			close(c.drained)
		}
	}
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
		return errors.E(ctx, errors.Unavailable, ErrClosed)
	default:
	}

	// If not waiting, just check current state
	if !waitForReady {
		if !c.IsReady() {
			if err := c.Err(); err != nil {
				return err
			}
			return errors.E(ctx, errors.Unavailable, ErrClosed)
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
			return errors.E(ctx, errors.Unavailable, ErrClosed)
		}
		return nil
	case <-c.closed:
		if err := c.Err(); err != nil {
			return err
		}
		return errors.E(ctx, errors.Unavailable, ErrClosed)
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

		// If packing is enabled, read and unpack the message.
		if c.packingEnabled {
			if err := c.readPackedMsg(msg); err != nil {
				if errors.Is(err, io.EOF) {
					c.setFatalError(io.EOF)
					return
				}
				c.setFatalError(errors.E(c.ctx, errors.Internal, fmt.Errorf("read error: %w", err)))
				return
			}
		} else {
			if _, err := msg.UnmarshalReader(c.transport); err != nil {
				if errors.Is(err, io.EOF) {
					c.setFatalError(io.EOF)
					return
				}
				c.setFatalError(errors.E(c.ctx, errors.Internal, fmt.Errorf("read error: %w", err)))
				return
			}
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

// readPackedMsg reads a packed message from the transport and unmarshals it.
func (c *Conn) readPackedMsg(msg msgs.Msg) error {
	// Read pack header (16 bytes: 8 bytes unpacked size + 8 bytes packed size).
	header := make([]byte, pack.HeaderSize)
	if _, err := io.ReadFull(c.transport, header); err != nil {
		return err
	}

	packedSize := int(binary.LittleEndian.Uint64(header[8:16]))

	// Read packed data.
	packedData := make([]byte, pack.HeaderSize+packedSize)
	copy(packedData, header)
	if _, err := io.ReadFull(c.transport, packedData[pack.HeaderSize:]); err != nil {
		return err
	}

	// Unpack.
	unpacked, err := pack.Unpack(c.ctx, packedData)
	if err != nil {
		return errors.E(c.ctx, errors.Internal, fmt.Errorf("unpack error: %w", err))
	}
	defer unpacked.Release(c.ctx)

	// Unmarshal from unpacked bytes.
	if _, err := msg.UnmarshalReader(bytes.NewReader(unpacked.Bytes())); err != nil {
		return err
	}

	return nil
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
		c.checkDrained()
		close(sess.readyCh) // Signal that session setup is complete (even though it failed)
		sess.close()
		c.mu.Unlock()
		return
	}

	sess.id = ack.SessionID()

	// Negotiate packing on first successful OpenAck.
	if !c.packingNegotiated {
		c.packingNegotiated = true
		c.packingEnabled = c.usePacking && ack.Packing()
	}

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
	c.checkDrained()
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
	if c.maxRecvMsgSize > 0 && len(p.Payload()) > c.maxRecvMsgSize {
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
	c.setFatalError(errors.E(c.ctx, errors.Unavailable, fmt.Errorf("server going away: %s", ga.DebugData())))
}

// sendMsg sends a message on the transport.
// If packing is negotiated and enabled, the message is packed before sending
// (except for Open messages which are never packed).
func (c *Conn) sendMsg(msg msgs.Msg) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	select {
	case <-c.closed:
		return errors.E(c.ctx, errors.Unavailable, ErrClosed)
	default:
	}

	// Pack if enabled (Open messages are never packed since they negotiate packing).
	if c.packingEnabled && msg.Type() != msgs.TOpen {
		var buf bytes.Buffer
		if _, err := msg.MarshalWriter(&buf); err != nil {
			c.setFatalError(errors.E(c.ctx, errors.Internal, fmt.Errorf("marshal error: %w", err)))
			return err
		}

		packed, err := pack.Pack(c.ctx, buf.Bytes())
		if err != nil {
			c.setFatalError(errors.E(c.ctx, errors.Internal, fmt.Errorf("pack error: %w", err)))
			return err
		}
		defer packed.Release(c.ctx)

		if _, err := c.transport.Write(packed.Bytes()); err != nil {
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

// openSession opens a new session with the server.
func (c *Conn) openSession(ctx context.Context, pkg, service, call string, rpcType msgs.RPCType, md metadata.MD) (*session, error) {
	c.mu.Lock()
	select {
	case <-c.closed:
		c.mu.Unlock()
		return nil, errors.E(ctx, errors.Unavailable, ErrClosed)
	default:
	}

	// Reject new sessions if draining
	if c.draining {
		c.mu.Unlock()
		return nil, errors.E(ctx, errors.Unavailable, ErrDraining)
	}

	openID := c.nextOpenID
	c.nextOpenID++

	sess := newSession(0, rpcType)
	c.pending[openID] = sess
	c.mu.Unlock()

	// Build the Open message.
	open := msgs.NewOpenFromRaw(ctx, msgs.OpenRaw{
		OpenID: openID,
		Descr: &msgs.DescrRaw{
			Package: pkg,
			Service: service,
			Call:    call,
			Type:    rpcType,
		},
		ProtocolMajor:  ProtocolMajor,
		ProtocolMinor:  ProtocolMinor,
		MaxPayloadSize: c.maxPayloadSize,
		Packing:        c.usePacking,
	})

	// Set deadline if context has one.
	if deadline, ok := ctx.Deadline(); ok {
		if ms := time.Until(deadline).Milliseconds(); ms > 0 {
			open = open.SetDeadlineMS(uint64(ms))
		}
	}

	// Add metadata if provided.
	if md != nil {
		open.MetadataAppend(ctx, md.ToMsgs(ctx)...)
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
		return nil, errors.E(ctx, errors.Unavailable, ErrClosed)
	case <-timeout.C:
		c.mu.Lock()
		delete(c.pending, openID)
		c.mu.Unlock()
		return nil, errors.E(ctx, errors.DeadlineExceeded, ErrTimeout)
	case <-sess.readyCh:
		// OpenAck was received. Check if accepted or rejected.
		// If sess.id != 0, the session was accepted and added to sessions map.
		if sess.id != 0 {
			return sess, nil
		}
		// sess.id == 0 means the open was rejected.
		return nil, errors.E(ctx, errors.Unavailable, errors.New("session open rejected by server"))
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
		return errors.E(c.ctx, errors.ResourceExhausted, fmt.Errorf("%w: %d bytes exceeds send limit of %d", ErrMessageTooLarge, len(payload), c.maxSendMsgSize))
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
		return nil, errors.E(ctx, errors.Unauthenticated, ErrInsecureTransport)
	}

	// Get credential metadata.
	credMD, err := opts.creds.GetRequestMetadata(ctx, uri)
	if err != nil {
		return nil, errors.E(ctx, errors.Unauthenticated, fmt.Errorf("failed to get credentials: %w", err))
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

// applyServiceConfig applies service configuration to the context and call options.
// It returns the potentially modified context (with timeout) and a cancel func if a
// timeout was applied (caller should defer cancel if non-nil).
func (c *Conn) applyServiceConfig(ctx context.Context, pkg, service, call string, callOpts *callOptions) (context.Context, context.CancelFunc) {
	if c.serviceConfig == nil {
		return ctx, nil
	}

	cfg, ok := c.serviceConfig.GetMethodConfig(pkg, service, call)
	if !ok {
		return ctx, nil
	}

	// Apply WaitForReady from config if not explicitly set via call option.
	// Note: We can't tell if waitForReady was explicitly set to false or just not set,
	// so service config WaitForReady only applies if the call option is false.
	if cfg.WaitForReady && !callOpts.waitForReady {
		callOpts.waitForReady = true
	}

	// Apply timeout if context doesn't already have a deadline.
	if cfg.Timeout > 0 {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			return context.WithTimeout(ctx, cfg.Timeout)
		}
	}

	return ctx, nil
}

// Sync creates a new synchronous RPC client.
// If a service config timeout is set for this method and the context doesn't
// have a deadline, the timeout is applied to all operations on this session.
func (c *Conn) Sync(ctx context.Context, pkg, service, call string, opts ...CallOption) (*SyncClient, error) {
	var callOpts callOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// Apply service config (timeout, wait-for-ready).
	// Note: We don't defer cancel() here because the context is used for the
	// lifetime of the SyncClient. The context will be cancelled when the
	// timeout expires or when the client is closed.
	ctx, _ = c.applyServiceConfig(ctx, pkg, service, call, &callOpts)

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
// If a service config timeout is set for this method and the context doesn't
// have a deadline, the timeout is applied to the stream lifetime.
func (c *Conn) BiDir(ctx context.Context, pkg, service, call string, opts ...CallOption) (*BiDirClient, error) {
	var callOpts callOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// Apply service config (timeout, wait-for-ready).
	ctx, _ = c.applyServiceConfig(ctx, pkg, service, call, &callOpts)

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
// If a service config timeout is set for this method and the context doesn't
// have a deadline, the timeout is applied to the stream lifetime.
func (c *Conn) Send(ctx context.Context, pkg, service, call string, opts ...CallOption) (*SendClient, error) {
	var callOpts callOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// Apply service config (timeout, wait-for-ready).
	ctx, _ = c.applyServiceConfig(ctx, pkg, service, call, &callOpts)

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
// If a service config timeout is set for this method and the context doesn't
// have a deadline, the timeout is applied to the stream lifetime.
func (c *Conn) Recv(ctx context.Context, pkg, service, call string, opts ...CallOption) (*RecvClient, error) {
	var callOpts callOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// Apply service config (timeout, wait-for-ready).
	ctx, _ = c.applyServiceConfig(ctx, pkg, service, call, &callOpts)

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
				*errPtr = errors.E(ctx, errors.Category(cl.ErrCode()), fmt.Errorf("server error: %s", cl.Error()))
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
				c.setFatalError(errors.E(c.ctx, errors.DeadlineExceeded, fmt.Errorf("keepalive timeout: no pong received within %v", c.pingTimeout)))
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
