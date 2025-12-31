package retry

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()
	if p.MaxAttempts != 3 {
		t.Errorf("[TestDefaultPolicy]: MaxAttempts = %d, want 3", p.MaxAttempts)
	}
	if p.InitialBackoff != 100*time.Millisecond {
		t.Errorf("[TestDefaultPolicy]: InitialBackoff = %v, want 100ms", p.InitialBackoff)
	}
	if p.MaxBackoff != 5*time.Second {
		t.Errorf("[TestDefaultPolicy]: MaxBackoff = %v, want 5s", p.MaxBackoff)
	}
	if p.Multiplier != 2.0 {
		t.Errorf("[TestDefaultPolicy]: Multiplier = %f, want 2.0", p.Multiplier)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    bool
	}{
		{
			name: "Success: nil error",
			err:  nil,
			want: false,
		},
		{
			name: "Success: internal error is retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrInternal.String()),
			want: true,
		},
		{
			name: "Success: unavailable error is retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrUnavailable.String()),
			want: true,
		},
		{
			name: "Success: resource exhausted is retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrResourceExhausted.String()),
			want: true,
		},
		{
			name: "Success: aborted is retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrAborted.String()),
			want: true,
		},
		{
			name: "Success: deadline exceeded is not retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrDeadlineExceeded.String()),
			want: false,
		},
		{
			name: "Success: canceled is not retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrCanceled.String()),
			want: false,
		},
		{
			name: "Success: invalid argument is not retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrInvalidArgument.String()),
			want: false,
		},
		{
			name: "Success: not found is not retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrNotFound.String()),
			want: false,
		},
		{
			name: "Success: permission denied is not retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrPermissionDenied.String()),
			want: false,
		},
		{
			name: "Success: unauthenticated is not retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrUnauthenticated.String()),
			want: false,
		},
		{
			name: "Success: unimplemented is not retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrUnimplemented.String()),
			want: false,
		},
		{
			name: "Success: already exists is not retryable",
			err:  fmt.Errorf("rpc error: %s", msgs.ErrAlreadyExists.String()),
			want: false,
		},
		{
			name: "Success: unknown error is not retryable",
			err:  errors.New("some unknown error"),
			want: false,
		},
	}

	for _, test := range tests {
		got := IsRetryable(test.err)
		if got != test.want {
			t.Errorf("[TestIsRetryable](%s): got %v, want %v", test.name, got, test.want)
		}
	}
}

func TestUnaryClientInterceptorNoRetry(t *testing.T) {
	policy := Policy{MaxAttempts: 0}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	calls := 0
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls++
		return []byte("response"), nil
	}

	resp, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err != nil {
		t.Errorf("[TestUnaryClientInterceptorNoRetry]: got err = %v, want nil", err)
	}
	if string(resp) != "response" {
		t.Errorf("[TestUnaryClientInterceptorNoRetry]: got resp = %q, want %q", resp, "response")
	}
	if calls != 1 {
		t.Errorf("[TestUnaryClientInterceptorNoRetry]: got calls = %d, want 1", calls)
	}
}

func TestUnaryClientInterceptorSuccess(t *testing.T) {
	policy := Policy{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	calls := 0
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls++
		return []byte("response"), nil
	}

	resp, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err != nil {
		t.Errorf("[TestUnaryClientInterceptorSuccess]: got err = %v, want nil", err)
	}
	if string(resp) != "response" {
		t.Errorf("[TestUnaryClientInterceptorSuccess]: got resp = %q, want %q", resp, "response")
	}
	if calls != 1 {
		t.Errorf("[TestUnaryClientInterceptorSuccess]: got calls = %d, want 1 (should succeed on first try)", calls)
	}
}

func TestUnaryClientInterceptorRetry(t *testing.T) {
	policy := Policy{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	calls := 0
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls++
		if calls < 3 {
			return nil, fmt.Errorf("rpc error: %s", msgs.ErrUnavailable.String())
		}
		return []byte("response"), nil
	}

	resp, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err != nil {
		t.Errorf("[TestUnaryClientInterceptorRetry]: got err = %v, want nil", err)
	}
	if string(resp) != "response" {
		t.Errorf("[TestUnaryClientInterceptorRetry]: got resp = %q, want %q", resp, "response")
	}
	if calls != 3 {
		t.Errorf("[TestUnaryClientInterceptorRetry]: got calls = %d, want 3 (should retry twice before success)", calls)
	}
}

func TestUnaryClientInterceptorMaxRetries(t *testing.T) {
	policy := Policy{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	calls := 0
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls++
		return nil, fmt.Errorf("rpc error: %s", msgs.ErrUnavailable.String())
	}

	_, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err == nil {
		t.Errorf("[TestUnaryClientInterceptorMaxRetries]: got err = nil, want error")
	}
	// MaxAttempts=3 means original call + 3 retries = 4 total calls
	if calls != 4 {
		t.Errorf("[TestUnaryClientInterceptorMaxRetries]: got calls = %d, want 4 (original + 3 retries)", calls)
	}
}

func TestUnaryClientInterceptorNonRetryable(t *testing.T) {
	policy := Policy{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	calls := 0
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls++
		return nil, fmt.Errorf("rpc error: %s", msgs.ErrInvalidArgument.String())
	}

	_, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err == nil {
		t.Errorf("[TestUnaryClientInterceptorNonRetryable]: got err = nil, want error")
	}
	if calls != 1 {
		t.Errorf("[TestUnaryClientInterceptorNonRetryable]: got calls = %d, want 1 (should not retry non-retryable errors)", calls)
	}
}

func TestUnaryClientInterceptorContextCanceled(t *testing.T) {
	policy := Policy{
		MaxAttempts:    5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
		Multiplier:     2.0,
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls++
		if calls == 2 {
			cancel() // Cancel context after second call
		}
		return nil, fmt.Errorf("rpc error: %s", msgs.ErrUnavailable.String())
	}

	_, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err == nil {
		t.Errorf("[TestUnaryClientInterceptorContextCanceled]: got err = nil, want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("[TestUnaryClientInterceptorContextCanceled]: got err = %v, want context.Canceled", err)
	}
}

func TestUnaryClientInterceptorCustomRetryable(t *testing.T) {
	policy := Policy{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		Retryable: func(err error) bool {
			return err.Error() == "custom-retryable"
		},
	}
	interceptor := UnaryClientInterceptor(policy)

	ctx := t.Context()
	calls := 0
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		calls++
		if calls < 3 {
			return nil, errors.New("custom-retryable")
		}
		return []byte("response"), nil
	}

	resp, err := interceptor(ctx, "test/method", []byte("req"), invoker)
	if err != nil {
		t.Errorf("[TestUnaryClientInterceptorCustomRetryable]: got err = %v, want nil", err)
	}
	if string(resp) != "response" {
		t.Errorf("[TestUnaryClientInterceptorCustomRetryable]: got resp = %q, want %q", resp, "response")
	}
	if calls != 3 {
		t.Errorf("[TestUnaryClientInterceptorCustomRetryable]: got calls = %d, want 3", calls)
	}
}
