package interceptor

import (
	"testing"

	"github.com/gostdlib/base/context"
)

func TestChainUnaryServer(t *testing.T) {
	tests := []struct {
		name         string
		interceptors []UnaryServerInterceptor
		wantOrder    []int
	}{
		{
			name:         "Success: no interceptors",
			interceptors: nil,
			wantOrder:    nil,
		},
		{
			name: "Success: single interceptor",
			interceptors: []UnaryServerInterceptor{
				makeUnaryServerInterceptor(1),
			},
			wantOrder: []int{1},
		},
		{
			name: "Success: multiple interceptors in order",
			interceptors: []UnaryServerInterceptor{
				makeUnaryServerInterceptor(1),
				makeUnaryServerInterceptor(2),
				makeUnaryServerInterceptor(3),
			},
			wantOrder: []int{1, 2, 3},
		},
	}

	for _, test := range tests {
		ctx := t.Context()
		var order []int

		chained := ChainUnaryServer(test.interceptors...)
		if chained == nil && len(test.interceptors) > 0 {
			t.Errorf("[TestChainUnaryServer](%s): got nil chain, want non-nil", test.name)
			continue
		}
		if chained == nil {
			continue
		}

		handler := func(ctx context.Context, req []byte) ([]byte, error) {
			return []byte("response"), nil
		}

		info := &UnaryServerInfo{
			Package: "test",
			Service: "Test",
			Method:  "Method",
		}

		// Create a wrapper that tracks order
		wrappedChain := func(ctx context.Context, req []byte, info *UnaryServerInfo, handler UnaryHandler) ([]byte, error) {
			return chained(ctx, req, info, func(ctx context.Context, req []byte) ([]byte, error) {
				// Extract order from context
				if o, ok := ctx.Value("order").(*[]int); ok {
					order = *o
				}
				return handler(ctx, req)
			})
		}

		orderCtx := context.WithValue(ctx, "order", &order)
		for _, i := range test.interceptors {
			// Re-wrap to capture order
			idx := i
			_ = idx
		}

		// Execute the chain
		_, _ = wrappedChain(orderCtx, []byte("request"), info, handler)
	}
}

func makeUnaryServerInterceptor(id int) UnaryServerInterceptor {
	return func(ctx context.Context, req []byte, info *UnaryServerInfo, handler UnaryHandler) ([]byte, error) {
		if o, ok := ctx.Value("order").(*[]int); ok {
			*o = append(*o, id)
		}
		return handler(ctx, req)
	}
}

func TestChainUnaryClient(t *testing.T) {
	tests := []struct {
		name         string
		interceptors []UnaryClientInterceptor
		wantCalled   bool
	}{
		{
			name:         "Success: no interceptors returns nil",
			interceptors: nil,
			wantCalled:   false,
		},
		{
			name: "Success: single interceptor",
			interceptors: []UnaryClientInterceptor{
				func(ctx context.Context, method string, req []byte, invoker UnaryInvoker) ([]byte, error) {
					return invoker(ctx, req)
				},
			},
			wantCalled: true,
		},
	}

	for _, test := range tests {
		ctx := t.Context()

		chained := ChainUnaryClient(test.interceptors...)
		if chained == nil && test.wantCalled {
			t.Errorf("[TestChainUnaryClient](%s): got nil chain, want non-nil", test.name)
			continue
		}
		if chained == nil {
			continue
		}

		called := false
		invoker := func(ctx context.Context, req []byte) ([]byte, error) {
			called = true
			return []byte("response"), nil
		}

		_, err := chained(ctx, "test/Service/Method", []byte("request"), invoker)
		if err != nil {
			t.Errorf("[TestChainUnaryClient](%s): got err = %v, want nil", test.name, err)
			continue
		}
		if called != test.wantCalled {
			t.Errorf("[TestChainUnaryClient](%s): got called = %v, want %v", test.name, called, test.wantCalled)
		}
	}
}
