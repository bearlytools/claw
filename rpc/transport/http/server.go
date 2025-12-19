package http

import (
	"errors"
	"io"
	"net"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/bearlytools/claw/rpc/server"
	"github.com/bearlytools/claw/rpc/transport"
)

// Handler is an http.Handler that accepts RPC connections over HTTP.
// It creates a bidirectional streaming connection using chunked transfer encoding.
type Handler struct {
	server    *server.Server
	config    *config
	onConnect func(transport.Transport)
}

// NewHandler creates an HTTP handler that serves RPC connections.
// The handler can be mounted on any HTTP router/mux.
//
// Example:
//
//	srv := server.New()
//	// ... register handlers ...
//	handler := http.NewHandler(srv)
//	httpServer := &http.Server{Addr: ":8080", Handler: handler}
//	httpServer.ListenAndServe()
func NewHandler(srv *server.Server, opts ...Option) *Handler {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &Handler{
		server: srv,
		config: cfg,
	}
}

// OnConnect sets a callback that is called when a new connection is established.
// This can be used for logging, metrics, or connection tracking.
func (h *Handler) OnConnect(fn func(transport.Transport)) {
	h.onConnect = fn
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate method.
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST for RPC connections.", http.StatusMethodNotAllowed)
		return
	}

	// Validate content type.
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && contentType != ContentType {
		http.Error(w, "Unsupported content type. Use "+ContentType, http.StatusUnsupportedMediaType)
		return
	}

	// Check for streaming support (http.Flusher is required).
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported by server", http.StatusInternalServerError)
		return
	}

	// Set response headers for streaming.
	w.Header().Set("Content-Type", ContentType)
	w.Header().Set(ProtocolVersionHeader, ProtocolVersion)
	// Note: Don't set Transfer-Encoding manually, Go handles chunked encoding.
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Determine addresses for the transport.
	localAddr := h.localAddr(r)
	remoteAddr := h.remoteAddr(r)

	// Create the bidirectional transport.
	// Reader: request body (client -> server)
	// Writer: response writer with flushing (server -> client)
	trans := newServerTransport(r.Body, w, flusher, localAddr, remoteAddr)

	// Notify callback if set.
	if h.onConnect != nil {
		h.onConnect(trans)
	}

	// Serve the RPC connection.
	// This blocks until the connection is closed.
	ctx := r.Context()
	err := h.server.Serve(ctx, trans)

	// Close the transport before returning to ensure no writes happen
	// after the handler completes (ResponseWriter becomes invalid).
	trans.Close()

	if err != nil && !errors.Is(err, io.EOF) {
		// Log error if needed, but don't write to response
		// as headers are already sent.
		_ = err
	}
}

// localAddr returns the local address for the connection.
func (h *Handler) localAddr(r *http.Request) net.Addr {
	// Try to get the local address from the request context.
	if localAddr, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
		return localAddr
	}
	// Fall back to a generic HTTP address.
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return &httpAddr{network: scheme, addr: r.Host}
}

// remoteAddr returns the remote address for the connection.
func (h *Handler) remoteAddr(r *http.Request) net.Addr {
	// Use X-Forwarded-For if present (for proxied connections).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return &httpAddr{network: "tcp", addr: xff}
	}
	// Fall back to RemoteAddr.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return &httpAddr{network: "tcp", addr: r.RemoteAddr}
	}
	return &httpAddr{network: "tcp", addr: host}
}

// Path returns the configured path for this handler.
// This is informational and does not affect routing.
func (h *Handler) Path() string {
	return h.config.path
}

// H2CHandler returns an http.Handler that supports h2c (HTTP/2 cleartext).
// This enables HTTP/2 bidirectional streaming without TLS.
//
// Example:
//
//	srv := server.New()
//	handler := http.NewHandler(srv)
//	httpServer := &http.Server{
//	    Addr:    ":8080",
//	    Handler: handler.H2CHandler(),
//	}
//	httpServer.ListenAndServe()
func (h *Handler) H2CHandler() http.Handler {
	h2s := &http2.Server{}
	return h2c.NewHandler(h, h2s)
}
