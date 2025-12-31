package interceptor

import (
	"github.com/gostdlib/base/context"
)

// ChainUnaryServer chains multiple unary server interceptors into one.
// Interceptors are executed in the order provided.
func ChainUnaryServer(interceptors ...UnaryServerInterceptor) UnaryServerInterceptor {
	switch len(interceptors) {
	case 0:
		return nil
	case 1:
		return interceptors[0]
	}

	return func(ctx context.Context, req []byte, info *UnaryServerInfo, handler UnaryHandler) ([]byte, error) {
		return chainUnaryServerHandler(interceptors, 0, info, handler)(ctx, req)
	}
}

func chainUnaryServerHandler(interceptors []UnaryServerInterceptor, idx int, info *UnaryServerInfo, finalHandler UnaryHandler) UnaryHandler {
	if idx == len(interceptors) {
		return finalHandler
	}
	return func(ctx context.Context, req []byte) ([]byte, error) {
		return interceptors[idx](ctx, req, info, chainUnaryServerHandler(interceptors, idx+1, info, finalHandler))
	}
}

// ChainStreamServer chains multiple stream server interceptors into one.
// Interceptors are executed in the order provided.
func ChainStreamServer(interceptors ...StreamServerInterceptor) StreamServerInterceptor {
	switch len(interceptors) {
	case 0:
		return nil
	case 1:
		return interceptors[0]
	}

	return func(ctx context.Context, stream ServerStream, info *StreamServerInfo, handler StreamHandler) error {
		return chainStreamServerHandler(interceptors, 0, info, handler)(ctx, stream)
	}
}

func chainStreamServerHandler(interceptors []StreamServerInterceptor, idx int, info *StreamServerInfo, finalHandler StreamHandler) StreamHandler {
	if idx == len(interceptors) {
		return finalHandler
	}
	return func(ctx context.Context, stream ServerStream) error {
		return interceptors[idx](ctx, stream, info, chainStreamServerHandler(interceptors, idx+1, info, finalHandler))
	}
}

// ChainUnaryClient chains multiple unary client interceptors into one.
// Interceptors are executed in the order provided.
func ChainUnaryClient(interceptors ...UnaryClientInterceptor) UnaryClientInterceptor {
	switch len(interceptors) {
	case 0:
		return nil
	case 1:
		return interceptors[0]
	}

	return func(ctx context.Context, method string, req []byte, invoker UnaryInvoker) ([]byte, error) {
		return chainUnaryClientInvoker(interceptors, 0, method, invoker)(ctx, req)
	}
}

func chainUnaryClientInvoker(interceptors []UnaryClientInterceptor, idx int, method string, finalInvoker UnaryInvoker) UnaryInvoker {
	if idx == len(interceptors) {
		return finalInvoker
	}
	return func(ctx context.Context, req []byte) ([]byte, error) {
		return interceptors[idx](ctx, method, req, chainUnaryClientInvoker(interceptors, idx+1, method, finalInvoker))
	}
}

// ChainStreamClient chains multiple stream client interceptors into one.
// Interceptors are executed in the order provided.
func ChainStreamClient(interceptors ...StreamClientInterceptor) StreamClientInterceptor {
	switch len(interceptors) {
	case 0:
		return nil
	case 1:
		return interceptors[0]
	}

	return func(ctx context.Context, method string, streamer ClientStreamer) (ClientStream, error) {
		return chainStreamClientStreamer(interceptors, 0, method, streamer)(ctx)
	}
}

func chainStreamClientStreamer(interceptors []StreamClientInterceptor, idx int, method string, finalStreamer ClientStreamer) ClientStreamer {
	if idx == len(interceptors) {
		return finalStreamer
	}
	return func(ctx context.Context) (ClientStream, error) {
		return interceptors[idx](ctx, method, chainStreamClientStreamer(interceptors, idx+1, method, finalStreamer))
	}
}
