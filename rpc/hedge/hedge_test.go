package hedge

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()
	if p.MaxHedgedRequests != 1 {
		t.Errorf("[TestDefaultPolicy]: MaxHedgedRequests = %d, want 1", p.MaxHedgedRequests)
	}
	if p.HedgeDelay != 50*time.Millisecond {
		t.Errorf("[TestDefaultPolicy]: HedgeDelay = %v, want 50ms", p.HedgeDelay)
	}
}

func TestUnaryClientInterceptorDisabled(t *testing.T) {
	policy := Policy{MaxHedgedRequests: 0}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	calls := 0
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls++
		return []byte("response"), nil
	}

	resp, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err != nil {
		t.Errorf("[TestUnaryClientInterceptorDisabled]: got err = %v, want nil", err)
	}
	if string(resp) != "response" {
		t.Errorf("[TestUnaryClientInterceptorDisabled]: got resp = %q, want %q", resp, "response")
	}
	if calls != 1 {
		t.Errorf("[TestUnaryClientInterceptorDisabled]: got calls = %d, want 1", calls)
	}
}

func TestUnaryClientInterceptorSuccess(t *testing.T) {
	policy := Policy{
		MaxHedgedRequests: 2,
		HedgeDelay:        10 * time.Millisecond,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	var calls atomic.Int32
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls.Add(1)
		return []byte("response"), nil
	}

	resp, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err != nil {
		t.Errorf("[TestUnaryClientInterceptorSuccess]: got err = %v, want nil", err)
	}
	if string(resp) != "response" {
		t.Errorf("[TestUnaryClientInterceptorSuccess]: got resp = %q, want %q", resp, "response")
	}
	// Original request should succeed immediately, hedges may or may not start
	if calls.Load() < 1 {
		t.Errorf("[TestUnaryClientInterceptorSuccess]: got calls = %d, want >= 1", calls.Load())
	}
}

func TestUnaryClientInterceptorHedgeWins(t *testing.T) {
	policy := Policy{
		MaxHedgedRequests: 1,
		HedgeDelay:        5 * time.Millisecond,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	var calls atomic.Int32
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		n := calls.Add(1)
		if n == 1 {
			// First call (original) is slow
			time.Sleep(50 * time.Millisecond)
		}
		// Second call (hedge) is fast
		return []byte(fmt.Sprintf("response-%d", n)), nil
	}

	resp, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err != nil {
		t.Errorf("[TestUnaryClientInterceptorHedgeWins]: got err = %v, want nil", err)
	}
	// Hedge (call 2) should win since it's faster
	if string(resp) != "response-2" {
		t.Errorf("[TestUnaryClientInterceptorHedgeWins]: got resp = %q, want %q (hedge should win)", resp, "response-2")
	}
}

func TestUnaryClientInterceptorAllFail(t *testing.T) {
	policy := Policy{
		MaxHedgedRequests: 2,
		HedgeDelay:        5 * time.Millisecond,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	var calls atomic.Int32
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls.Add(1)
		return nil, fmt.Errorf("rpc error: %s", msgs.ErrUnavailable.String())
	}

	_, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err == nil {
		t.Errorf("[TestUnaryClientInterceptorAllFail]: got err = nil, want error")
	}
	// Should have 3 total calls (1 original + 2 hedges)
	// Give some time for all hedges to complete
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != 3 {
		t.Errorf("[TestUnaryClientInterceptorAllFail]: got calls = %d, want 3", calls.Load())
	}
}

func TestUnaryClientInterceptorFatalError(t *testing.T) {
	policy := Policy{
		MaxHedgedRequests: 2,
		HedgeDelay:        10 * time.Millisecond,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	var calls atomic.Int32
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls.Add(1)
		// Return fatal error immediately
		return nil, fmt.Errorf("rpc error: %s", msgs.ErrInvalidArgument.String())
	}

	_, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err == nil {
		t.Errorf("[TestUnaryClientInterceptorFatalError]: got err = nil, want error")
	}
	// Should return immediately on fatal error, not wait for hedges
	if calls.Load() != 1 {
		t.Errorf("[TestUnaryClientInterceptorFatalError]: got calls = %d, want 1 (should fail fast on fatal error)", calls.Load())
	}
}

func TestUnaryClientInterceptorContextCanceled(t *testing.T) {
	policy := Policy{
		MaxHedgedRequests: 2,
		HedgeDelay:        100 * time.Millisecond,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx, cancel := context.WithCancel(t.Context())
	var calls atomic.Int32
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls.Add(1)
		// Block until cancelled
		<-ctx.Done()
		return nil, ctx.Err()
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err == nil {
		t.Errorf("[TestUnaryClientInterceptorContextCanceled]: got err = nil, want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("[TestUnaryClientInterceptorContextCanceled]: got err = %v, want context.Canceled", err)
	}
}

func TestUnaryClientInterceptorCancelsOthers(t *testing.T) {
	policy := Policy{
		MaxHedgedRequests: 2,
		HedgeDelay:        1 * time.Millisecond,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	var cancelled atomic.Int32
	var calls atomic.Int32

	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		n := calls.Add(1)
		if n == 1 {
			// First call succeeds quickly
			return []byte("response"), nil
		}
		// Other calls wait and track cancellation
		<-ctx.Done()
		cancelled.Add(1)
		return nil, ctx.Err()
	}

	resp, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err != nil {
		t.Errorf("[TestUnaryClientInterceptorCancelsOthers]: got err = %v, want nil", err)
	}
	if string(resp) != "response" {
		t.Errorf("[TestUnaryClientInterceptorCancelsOthers]: got resp = %q, want %q", resp, "response")
	}

	// Give time for other requests to be cancelled
	time.Sleep(20 * time.Millisecond)

	// Other hedges should have been cancelled
	if cancelled.Load() == 0 && calls.Load() > 1 {
		t.Errorf("[TestUnaryClientInterceptorCancelsOthers]: hedges should have been cancelled")
	}
}

func TestIsFatal(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Success: nil error",
			err:  nil,
			want: false,
		},
		{
			name: "Success: unavailable is not fatal",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrUnavailable.String()),
			want: false,
		},
		{
			name: "Success: internal is not fatal",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrInternal.String()),
			want: false,
		},
		{
			name: "Success: invalid argument is fatal",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrInvalidArgument.String()),
			want: true,
		},
		{
			name: "Success: not found is fatal",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrNotFound.String()),
			want: true,
		},
		{
			name: "Success: permission denied is fatal",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrPermissionDenied.String()),
			want: true,
		},
		{
			name: "Success: unauthenticated is fatal",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrUnauthenticated.String()),
			want: true,
		},
		{
			name: "Success: unimplemented is fatal",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrUnimplemented.String()),
			want: true,
		},
		{
			name: "Success: deadline exceeded is fatal",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrDeadlineExceeded.String()),
			want: true,
		},
		{
			name: "Success: canceled is fatal",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrCanceled.String()),
			want: true,
		},
		{
			name: "Success: context canceled is fatal",
			err:  errors.New("context canceled"),
			want: true,
		},
		{
			name: "Success: context deadline exceeded is fatal",
			err:  errors.New("context deadline exceeded"),
			want: true,
		},
		{
			name: "Success: unknown error is not fatal",
			err:  errors.New("some unknown error"),
			want: false,
		},
	}

	for _, test := range tests {
		got := isFatal(test.err, nil)
		if got != test.want {
			t.Errorf("[TestIsFatal](%s): got %v, want %v", test.name, got, test.want)
		}
	}
}

func TestIsFatalWithNonFatalCodes(t *testing.T) {
	// If NonFatalCodes is specified, only those codes are non-fatal
	nonFatalCodes := []msgs.ErrCode{msgs.ErrUnavailable}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Success: unavailable is non-fatal when in list",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrUnavailable.String()),
			want: false,
		},
		{
			name: "Success: internal is fatal when not in list",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrInternal.String()),
			want: true,
		},
	}

	for _, test := range tests {
		got := isFatal(test.err, nonFatalCodes)
		if got != test.want {
			t.Errorf("[TestIsFatalWithNonFatalCodes](%s): got %v, want %v", test.name, got, test.want)
		}
	}
}

func TestZeroPolicyDisablesHedging(t *testing.T) {
	// Zero value policy should disable hedging
	var policy Policy
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	calls := 0
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls++
		return []byte("response"), nil
	}

	resp, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err != nil {
		t.Errorf("[TestZeroPolicyDisablesHedging]: got err = %v, want nil", err)
	}
	if string(resp) != "response" {
		t.Errorf("[TestZeroPolicyDisablesHedging]: got resp = %q, want %q", resp, "response")
	}
	if calls != 1 {
		t.Errorf("[TestZeroPolicyDisablesHedging]: got calls = %d, want 1 (no hedging)", calls)
	}
}
