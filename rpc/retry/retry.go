// Package retry provides retry policies and interceptors for RPC calls.
package retry

import (
	"strings"
	"time"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// Policy configures retry behavior for RPC calls.
type Policy struct {
	// MaxAttempts is the maximum number of attempts (including the first call).
	// 0 means no retry (single attempt), 1 means retry once (2 total attempts).
	MaxAttempts int

	// InitialBackoff is the initial wait time before the first retry.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum wait time between retries.
	MaxBackoff time.Duration

	// Multiplier is the factor by which the backoff increases after each retry.
	Multiplier float64

	// Retryable is an optional function to determine if an error is retryable.
	// If nil, the default retryable check is used.
	Retryable func(err error) bool
}

// DefaultPolicy returns a sensible default retry policy.
// 3 attempts total, 100ms initial backoff, 5s max backoff, 2x multiplier.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		Multiplier:     2.0,
	}
}

// UnaryClientInterceptor returns a unary client interceptor that retries failed calls
// according to the provided policy.
func UnaryClientInterceptor(policy Policy) interceptor.UnaryClientInterceptor {
	if policy.MaxAttempts <= 0 {
		// No retries, just pass through.
		return func(ctx context.Context, method string, req []byte, invoker interceptor.UnaryInvoker) ([]byte, error) {
			return invoker(ctx, req)
		}
	}

	retryable := policy.Retryable
	if retryable == nil {
		retryable = isRetryable
	}

	return func(ctx context.Context, method string, req []byte, invoker interceptor.UnaryInvoker) ([]byte, error) {
		var lastErr error
		backoff := policy.InitialBackoff

		for attempt := 0; attempt <= policy.MaxAttempts; attempt++ {
			resp, err := invoker(ctx, req)
			if err == nil {
				return resp, nil
			}

			if !retryable(err) {
				return nil, err
			}
			lastErr = err

			// Don't wait after the last attempt.
			if attempt < policy.MaxAttempts {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
				}

				// Increase backoff for next attempt.
				backoff = time.Duration(float64(backoff) * policy.Multiplier)
				if backoff > policy.MaxBackoff {
					backoff = policy.MaxBackoff
				}
			}
		}
		return nil, lastErr
	}
}

// isRetryable determines if an error should be retried based on error codes.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for retryable error codes in the error message.
	// Errors from Close messages contain the error code name.

	// Don't retry deadline exceeded - the deadline has already passed.
	if strings.Contains(errStr, msgs.ErrDeadlineExceeded.String()) {
		return false
	}

	// Don't retry cancelled - the request was explicitly cancelled.
	if strings.Contains(errStr, msgs.ErrCanceled.String()) {
		return false
	}

	// Don't retry invalid arguments, not found, already exists, permission denied.
	if strings.Contains(errStr, msgs.ErrInvalidArgument.String()) ||
		strings.Contains(errStr, msgs.ErrNotFound.String()) ||
		strings.Contains(errStr, msgs.ErrAlreadyExists.String()) ||
		strings.Contains(errStr, msgs.ErrPermissionDenied.String()) ||
		strings.Contains(errStr, msgs.ErrUnauthenticated.String()) ||
		strings.Contains(errStr, msgs.ErrUnimplemented.String()) {
		return false
	}

	// Retry internal, unavailable, resource exhausted, and aborted.
	if strings.Contains(errStr, msgs.ErrInternal.String()) ||
		strings.Contains(errStr, msgs.ErrUnavailable.String()) ||
		strings.Contains(errStr, msgs.ErrResourceExhausted.String()) ||
		strings.Contains(errStr, msgs.ErrAborted.String()) {
		return true
	}

	// Default: don't retry unknown errors.
	return false
}

// IsRetryable checks if an error is retryable according to the default rules.
func IsRetryable(err error) bool {
	return isRetryable(err)
}
