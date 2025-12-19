// Package http provides HTTP transport for RPC connections.
package http

import (
	"io"
	"net"
	"net/http"

	"github.com/gostdlib/base/concurrency/sync"

	"github.com/bearlytools/claw/rpc/transport"
)

// ContentType is the MIME type for Claw RPC messages.
const ContentType = "application/x-claw-rpc"

// ProtocolVersionHeader is the header name for protocol version.
const ProtocolVersionHeader = "X-Claw-Protocol-Version"

// ProtocolVersion is the current protocol version.
const ProtocolVersion = "1.0"

// httpAddr implements net.Addr for HTTP connections.
type httpAddr struct {
	network string
	addr    string
}

func (a *httpAddr) Network() string { return a.network }
func (a *httpAddr) String() string  { return a.addr }

// serverTransport implements transport.Transport for server-side HTTP connections.
// It wraps the request body (for reading) and response writer (for writing).
type serverTransport struct {
	reader     io.ReadCloser
	writer     io.Writer
	flusher    http.Flusher
	localAddr  net.Addr
	remoteAddr net.Addr

	mu       sync.Mutex
	closed   bool
	closeErr error
}

// newServerTransport creates a new server-side transport.
func newServerTransport(r io.ReadCloser, w io.Writer, flusher http.Flusher, local, remote net.Addr) *serverTransport {
	return &serverTransport{
		reader:     r,
		writer:     w,
		flusher:    flusher,
		localAddr:  local,
		remoteAddr: remote,
	}
}

func (t *serverTransport) Read(p []byte) (int, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	t.mu.Unlock()

	return t.reader.Read(p)
}

func (t *serverTransport) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return 0, io.ErrClosedPipe
	}

	n, err := t.writer.Write(p)
	if err != nil {
		return n, err
	}

	// Flush after each write to ensure messages are sent immediately.
	if t.flusher != nil {
		t.flusher.Flush()
	}

	return n, nil
}

func (t *serverTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return t.closeErr
	}
	t.closed = true

	// Close the reader to signal end of input.
	t.closeErr = t.reader.Close()
	return t.closeErr
}

func (t *serverTransport) LocalAddr() net.Addr {
	return t.localAddr
}

func (t *serverTransport) RemoteAddr() net.Addr {
	return t.remoteAddr
}

// Verify serverTransport implements transport.Transport.
var _ transport.Transport = (*serverTransport)(nil)

// flushWriter wraps an io.Writer with an http.Flusher.
type flushWriter struct {
	w       io.Writer
	flusher http.Flusher
}

func (fw flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if err != nil {
		return n, err
	}
	fw.flusher.Flush()
	return n, nil
}
