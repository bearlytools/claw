// Package interceptor provides interceptor types for cross-cutting concerns
// like authentication, logging, metrics, and tracing in RPC calls.
package interceptor

import (
	"iter"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// UnaryServerInfo contains RPC metadata available to unary server interceptors.
type UnaryServerInfo struct {
	Package   string
	Service   string
	Method    string
	SessionID uint32
	Metadata  []msgs.Metadata
}

// UnaryHandler is the handler that a unary server interceptor wraps.
type UnaryHandler func(ctx context.Context, req []byte) ([]byte, error)

// UnaryServerInterceptor intercepts synchronous RPC calls on the server.
// It receives the request, RPC info, and the next handler in the chain.
// The interceptor can modify the request, call the handler, and modify the response.
type UnaryServerInterceptor func(ctx context.Context, req []byte, info *UnaryServerInfo, handler UnaryHandler) ([]byte, error)

// StreamServerInfo contains RPC metadata available to stream server interceptors.
type StreamServerInfo struct {
	Package   string
	Service   string
	Method    string
	SessionID uint32
	Metadata  []msgs.Metadata
	RPCType   msgs.RPCType
}

// ServerStream is the stream interface passed to stream server interceptors.
type ServerStream interface {
	Send(payload []byte) error
	Recv() iter.Seq[[]byte]
	Context() context.Context
}

// StreamHandler is the handler that a stream server interceptor wraps.
type StreamHandler func(ctx context.Context, stream ServerStream) error

// StreamServerInterceptor intercepts streaming RPC calls on the server.
// It receives the stream, RPC info, and the next handler in the chain.
type StreamServerInterceptor func(ctx context.Context, stream ServerStream, info *StreamServerInfo, handler StreamHandler) error

// UnaryInvoker performs the actual unary RPC call on the client side.
type UnaryInvoker func(ctx context.Context, req []byte) ([]byte, error)

// UnaryClientInterceptor intercepts synchronous RPC calls on the client.
// It receives the method name (in "package/service/method" format), request,
// and the invoker that performs the actual call.
type UnaryClientInterceptor func(ctx context.Context, method string, req []byte, invoker UnaryInvoker) ([]byte, error)

// ClientStream is the stream interface passed to stream client interceptors.
type ClientStream interface {
	Send(ctx context.Context, payload []byte) error
	Recv(ctx context.Context) iter.Seq[[]byte]
	CloseSend() error
	Close() error
	Err() error
}

// ClientStreamer creates a client stream.
type ClientStreamer func(ctx context.Context) (ClientStream, error)

// StreamClientInterceptor intercepts streaming RPC calls on the client.
// It receives the method name and the streamer that creates the actual stream.
type StreamClientInterceptor func(ctx context.Context, method string, streamer ClientStreamer) (ClientStream, error)
