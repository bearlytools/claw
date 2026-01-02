// Package hedge provides hedging (speculative retry) for RPC calls.
// Hedging sends the same request to multiple backends in parallel and
// uses whichever response arrives first, reducing tail latency.
//
// Hedging is disabled by default and must be explicitly enabled via
// WithHedgePolicy() or by setting MaxHedgedRequests > 0.
package hedge

import (
	"strings"
	"time"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// Policy configures hedging behavior. Zero value means hedging is disabled.
type Policy struct {
	// MaxHedgedRequests is the maximum number of hedged requests (excluding original).
	// 0 means no hedging (disabled). 1 means 1 hedge (2 total requests).
	// Recommended: 1-2 for most use cases.
	MaxHedgedRequests int

	// HedgeDelay is how long to wait before sending each hedge request.
	// Should be based on expected P50-P90 latency. Too short = wasted requests.
	// Too long = no benefit. Typical: 10-50ms.
	HedgeDelay time.Duration

	// NonFatalCodes are error codes that don't immediately fail the hedge.
	// If nil, all errors except context cancellation are non-fatal.
	NonFatalCodes []msgs.ErrCode
}

// result holds the response from a hedged request.
type result struct {
	resp []byte
	err  error
}

// UnaryClientInterceptor returns a unary client interceptor that hedges calls
// according to the provided policy.
//
// Hedging is disabled if MaxHedgedRequests <= 0.
//
// When enabled, the interceptor sends the original request immediately, then
// sends additional "hedge" requests after each HedgeDelay interval. The first
// successful response is returned and all other in-flight requests are cancelled.
func UnaryClientInterceptor(policy Policy) interceptor.UnaryClientInterceptor {
	if policy.MaxHedgedRequests <= 0 {
		// Hedging disabled - pass through.
		return func(ctx context.Context, method string, req []byte, invoker interceptor.UnaryInvoker) ([]byte, error) {
			return invoker(ctx, req)
		}
	}

	return func(ctx context.Context, method string, req []byte, invoker interceptor.UnaryInvoker) ([]byte, error) {
		// Total requests = original + hedges
		totalRequests := policy.MaxHedgedRequests + 1

		// Result channel for responses (buffered to avoid goroutine leaks)
		results := make(chan result, totalRequests)

		// Create cancellable context for all hedges
		hedgeCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Launch original request immediately
		pool := context.Pool(ctx)
		pool.Submit(ctx, func() {
			resp, err := invoker(hedgeCtx, req)
			select {
			case results <- result{resp, err}:
			case <-hedgeCtx.Done():
			}
		})

		// Launch hedged requests after delays
		for i := 0; i < policy.MaxHedgedRequests; i++ {
			delay := policy.HedgeDelay * time.Duration(i+1)
			pool.Submit(ctx, func() {
				// Wait for hedge delay
				select {
				case <-hedgeCtx.Done():
					return
				case <-time.After(delay):
				}

				// Check if we should still send (context might be cancelled)
				select {
				case <-hedgeCtx.Done():
					return
				default:
				}

				resp, err := invoker(hedgeCtx, req)
				select {
				case results <- result{resp, err}:
				case <-hedgeCtx.Done():
				}
			})
		}

		// Wait for first success or all failures
		var lastErr error
		received := 0

		for received < totalRequests {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case r := <-results:
				received++

				if r.err == nil {
					cancel() // Cancel other hedges
					return r.resp, nil
				}

				// Check if error is fatal
				if isFatal(r.err, policy.NonFatalCodes) {
					cancel()
					return nil, r.err
				}

				lastErr = r.err
			}
		}

		return nil, lastErr
	}
}

// isFatal returns true if the error should immediately fail the hedge
// without waiting for other responses.
func isFatal(err error, nonFatalCodes []msgs.ErrCode) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Context errors are always fatal
	if strings.Contains(errStr, "context canceled") ||
		strings.Contains(errStr, "context deadline exceeded") {
		return true
	}

	// These error codes are always fatal (non-retryable)
	fatalCodes := []msgs.ErrCode{
		msgs.ErrInvalidArgument,
		msgs.ErrNotFound,
		msgs.ErrAlreadyExists,
		msgs.ErrPermissionDenied,
		msgs.ErrUnauthenticated,
		msgs.ErrUnimplemented,
		msgs.ErrDeadlineExceeded,
		msgs.ErrCanceled,
	}

	for _, code := range fatalCodes {
		if strings.Contains(errStr, code.String()) {
			return true
		}
	}

	// If NonFatalCodes is specified, only those codes are non-fatal
	if len(nonFatalCodes) > 0 {
		for _, code := range nonFatalCodes {
			if strings.Contains(errStr, code.String()) {
				return false // Explicitly non-fatal
			}
		}
		// Not in non-fatal list, so it's fatal
		return true
	}

	// Default: transient errors are non-fatal
	return false
}

// DefaultPolicy returns a sensible default hedging policy.
// 1 hedge (2 total requests), 50ms delay.
func DefaultPolicy() Policy {
	return Policy{
		MaxHedgedRequests: 1,
		HedgeDelay:        50 * time.Millisecond,
	}
}
