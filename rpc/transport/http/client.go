package http

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/http2"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/retry/exponential"

	"github.com/bearlytools/claw/rpc/transport"
)

// Common errors.
var (
	ErrClosed       = errors.New("transport closed")
	ErrNotConnected = errors.New("not connected")
)

// ClientTransport implements transport.Transport over HTTP.
// It establishes a streaming HTTP connection using chunked transfer encoding.
type ClientTransport struct {
	url        *url.URL
	httpClient *http.Client
	config     *config
	backoff    *exponential.Backoff

	// Connection state.
	mu         sync.Mutex
	connected  bool
	closed     bool
	closedCh   chan struct{}
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	resp       *http.Response
	connErr    error

	// Context for the current connection.
	ctx    context.Context
	cancel context.CancelFunc
}

// Dial creates a new HTTP transport connection to the specified URL.
// The URL should be http:// or https:// with the RPC endpoint path.
//
// Example:
//
//	transport, err := http.Dial(ctx, "https://example.com/rpc")
//	if err != nil {
//	    return err
//	}
//	defer transport.Close()
//	conn := client.New(ctx, transport)
func Dial(ctx context.Context, rawURL string, opts ...Option) (*ClientTransport, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q, use http or https", parsedURL.Scheme)
	}

	// Create HTTP client if not provided.
	httpClient := cfg.httpClient
	if httpClient == nil {
		var transport http.RoundTripper

		if parsedURL.Scheme == "https" {
			// HTTPS: Use standard HTTP/2 with TLS.
			tlsConfig := cfg.tlsConfig
			if tlsConfig == nil {
				tlsConfig = &tls.Config{}
			}
			transport = &http.Transport{
				TLSClientConfig:    tlsConfig,
				DisableCompression: true,
				ForceAttemptHTTP2:  true,
			}
		} else {
			// HTTP: Use h2c (HTTP/2 cleartext) for bidirectional streaming.
			transport = &http2.Transport{
				AllowHTTP: true,
				DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					// Dial without TLS for h2c.
					var d net.Dialer
					return d.DialContext(ctx, network, addr)
				},
				DisableCompression: true,
			}
		}

		httpClient = &http.Client{Transport: transport}
	}

	// Create backoff for reconnection.
	backoff, err := exponential.New(exponential.WithPolicy(cfg.retryPolicy))
	if err != nil {
		return nil, fmt.Errorf("failed to create backoff: %w", err)
	}

	t := &ClientTransport{
		url:        parsedURL,
		httpClient: httpClient,
		config:     cfg,
		backoff:    backoff,
		closedCh:   make(chan struct{}),
	}

	// Establish initial connection.
	if err := t.connect(ctx); err != nil {
		return nil, err
	}

	return t, nil
}

// connect establishes the HTTP streaming connection.
func (t *ClientTransport) connect(ctx context.Context) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrClosed
	}

	// Clean up any existing connection.
	t.cleanupLocked()

	// Create a pipe for the request body.
	// Writes to pipeWriter will be sent as the request body.
	t.pipeReader, t.pipeWriter = io.Pipe()

	// Create cancellable context for this connection.
	t.ctx, t.cancel = context.WithCancel(ctx)

	t.mu.Unlock()

	// Build the request.
	req, err := http.NewRequestWithContext(t.ctx, http.MethodPost, t.url.String(), t.pipeReader)
	if err != nil {
		t.mu.Lock()
		t.cleanupLocked()
		t.mu.Unlock()
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers.
	req.Header.Set("Content-Type", ContentType)
	req.Header.Set(ProtocolVersionHeader, ProtocolVersion)
	// Copy custom headers.
	for k, v := range t.config.headers {
		req.Header[k] = v
	}

	// Start the request asynchronously to avoid blocking.
	// http.Client.Do() blocks until response headers are received,
	// but the server may be waiting to read from the request body first.
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)

	go func() {
		resp, err := t.httpClient.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	// Wait for either response or error.
	// Use a timeout to avoid waiting forever.
	select {
	case <-t.ctx.Done():
		t.mu.Lock()
		t.cleanupLocked()
		t.mu.Unlock()
		return t.ctx.Err()
	case err := <-errCh:
		t.mu.Lock()
		t.cleanupLocked()
		t.mu.Unlock()
		return fmt.Errorf("failed to connect: %w", err)
	case resp := <-respCh:
		// Check response status.
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
			t.mu.Lock()
			t.cleanupLocked()
			t.mu.Unlock()
			return fmt.Errorf("server returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		t.mu.Lock()
		t.resp = resp
		t.connected = true
		t.connErr = nil
		t.mu.Unlock()

		return nil
	}
}

// cleanupLocked cleans up the current connection. Must hold t.mu.
func (t *ClientTransport) cleanupLocked() {
	t.connected = false

	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}

	if t.pipeWriter != nil {
		t.pipeWriter.Close()
		t.pipeWriter = nil
	}

	if t.pipeReader != nil {
		t.pipeReader.Close()
		t.pipeReader = nil
	}

	if t.resp != nil {
		t.resp.Body.Close()
		t.resp = nil
	}
}

// Read reads data from the server.
func (t *ClientTransport) Read(p []byte) (int, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return 0, ErrClosed
	}
	if !t.connected {
		t.mu.Unlock()
		return 0, ErrNotConnected
	}
	resp := t.resp
	t.mu.Unlock()

	n, err := resp.Body.Read(p)
	if err != nil {
		t.mu.Lock()
		t.connErr = err
		t.mu.Unlock()
	}
	return n, err
}

// Write writes data to the server.
func (t *ClientTransport) Write(p []byte) (int, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return 0, ErrClosed
	}
	if !t.connected {
		t.mu.Unlock()
		return 0, ErrNotConnected
	}
	writer := t.pipeWriter
	t.mu.Unlock()

	n, err := writer.Write(p)
	if err != nil {
		t.mu.Lock()
		t.connErr = err
		t.mu.Unlock()
	}
	return n, err
}

// Close closes the transport.
func (t *ClientTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	close(t.closedCh)
	t.cleanupLocked()
	t.mu.Unlock()

	return nil
}

// LocalAddr returns the local network address.
func (t *ClientTransport) LocalAddr() net.Addr {
	// HTTP doesn't expose local address easily.
	return nil
}

// RemoteAddr returns the remote network address.
func (t *ClientTransport) RemoteAddr() net.Addr {
	return &httpAddr{
		network: t.url.Scheme,
		addr:    t.url.Host,
	}
}

// Reconnect attempts to reconnect with exponential backoff.
// This is called automatically on connection errors when using ReconnectingTransport.
func (t *ClientTransport) Reconnect(ctx context.Context) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrClosed
	}
	t.mu.Unlock()

	return t.backoff.Retry(ctx, func(retryCtx context.Context, r exponential.Record) error {
		return t.connect(retryCtx)
	})
}

// Err returns any connection error.
func (t *ClientTransport) Err() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connErr
}

// Connected returns true if the transport is connected.
func (t *ClientTransport) Connected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connected && !t.closed
}

// Verify ClientTransport implements transport.Transport.
var _ transport.Transport = (*ClientTransport)(nil)

// Dialer implements transport.Dialer for HTTP connections.
type Dialer struct {
	url    string
	config *config
}

// NewDialer creates a new HTTP dialer.
func NewDialer(url string, opts ...Option) *Dialer {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &Dialer{url: url, config: cfg}
}

// Dial establishes a new HTTP transport connection.
func (d *Dialer) Dial(ctx context.Context) (transport.Transport, error) {
	return Dial(ctx, d.url,
		WithHTTPClient(d.config.httpClient),
		WithTLSConfig(d.config.tlsConfig),
		WithHeaders(d.config.headers),
		WithRetryPolicy(d.config.retryPolicy),
		WithDialTimeout(d.config.dialTimeout),
	)
}

// Verify Dialer implements transport.Dialer.
var _ transport.Dialer = (*Dialer)(nil)
