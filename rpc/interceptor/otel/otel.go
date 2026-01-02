package otel

import (
	"time"

	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/telemetry/otel/trace/span"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/bearlytools/claw/languages/go/errors"
	"github.com/bearlytools/claw/rpc/interceptor"
)

// Interceptor holds the OTEL instrumentation state.
type Interceptor struct {
	cfg Config

	// Server metrics
	serverDuration     metric.Float64Histogram
	serverRequestCount metric.Int64Counter
	serverRequestSize  metric.Int64Histogram
	serverResponseSize metric.Int64Histogram

	// Client metrics
	clientDuration     metric.Float64Histogram
	clientRequestCount metric.Int64Counter
	clientRequestSize  metric.Int64Histogram
	clientResponseSize metric.Int64Histogram
}

// New creates a new OTEL Interceptor with the given configuration.
func New(ctx context.Context, cfg Config) (*Interceptor, error) {
	i := &Interceptor{cfg: cfg}

	if cfg.EnableMetrics {
		if err := i.initMetrics(ctx); err != nil {
			return nil, err
		}
	}

	// Compile trace rules if provided.
	if cfg.TraceRules != nil {
		if err := cfg.TraceRules.compile(); err != nil {
			return nil, err
		}
	}

	return i, nil
}

// initMetrics initializes the OTEL metric instruments.
func (i *Interceptor) initMetrics(ctx context.Context) error {
	var meter metric.Meter
	if i.cfg.MeterProvider != nil {
		meter = i.cfg.MeterProvider.Meter("claw-rpc")
	} else {
		meter = context.Meter(ctx)
	}

	var err error

	// Server metrics
	i.serverDuration, err = meter.Float64Histogram(
		"rpc.server.duration",
		metric.WithDescription("Duration of RPC server calls in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	i.serverRequestCount, err = meter.Int64Counter(
		"rpc.server.request_count",
		metric.WithDescription("Total number of RPC server requests"),
	)
	if err != nil {
		return err
	}

	if i.cfg.RecordPayloadSize {
		i.serverRequestSize, err = meter.Int64Histogram(
			"rpc.server.request_size",
			metric.WithDescription("Size of RPC server requests in bytes"),
			metric.WithUnit("By"),
		)
		if err != nil {
			return err
		}

		i.serverResponseSize, err = meter.Int64Histogram(
			"rpc.server.response_size",
			metric.WithDescription("Size of RPC server responses in bytes"),
			metric.WithUnit("By"),
		)
		if err != nil {
			return err
		}
	}

	// Client metrics
	i.clientDuration, err = meter.Float64Histogram(
		"rpc.client.duration",
		metric.WithDescription("Duration of RPC client calls in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	i.clientRequestCount, err = meter.Int64Counter(
		"rpc.client.request_count",
		metric.WithDescription("Total number of RPC client requests"),
	)
	if err != nil {
		return err
	}

	if i.cfg.RecordPayloadSize {
		i.clientRequestSize, err = meter.Int64Histogram(
			"rpc.client.request_size",
			metric.WithDescription("Size of RPC client requests in bytes"),
			metric.WithUnit("By"),
		)
		if err != nil {
			return err
		}

		i.clientResponseSize, err = meter.Int64Histogram(
			"rpc.client.response_size",
			metric.WithDescription("Size of RPC client responses in bytes"),
			metric.WithUnit("By"),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// UnaryServerInterceptor returns a unary server interceptor with tracing and metrics.
func (i *Interceptor) UnaryServerInterceptor() interceptor.UnaryServerInterceptor {
	return func(ctx context.Context, req []byte, info *interceptor.UnaryServerInfo, handler interceptor.UnaryHandler) ([]byte, error) {
		method := info.Package + "/" + info.Service + "/" + info.Method
		start := time.Now()

		// Start span if tracing is enabled.
		if i.cfg.EnableTracing {
			var sp span.Span
			ctx, sp = span.New(ctx,
				span.WithName(method),
				span.WithSpanStartOption(trace.WithSpanKind(trace.SpanKindServer)),
			)
			defer sp.End()

			// Add span attributes.
			sp.Span.SetAttributes(
				attribute.String("rpc.system", "claw"),
				attribute.String("rpc.service", info.Service),
				attribute.String("rpc.method", info.Method),
				attribute.String("rpc.package", info.Package),
				attribute.Int64("rpc.session_id", int64(info.SessionID)),
			)
		}

		// Record request size.
		if i.cfg.EnableMetrics && i.cfg.RecordPayloadSize && i.serverRequestSize != nil {
			i.serverRequestSize.Record(ctx, int64(len(req)),
				metric.WithAttributes(attribute.String("rpc_method", method)))
		}

		// Call handler.
		resp, err := handler(ctx, req)

		// Record metrics.
		if i.cfg.EnableMetrics {
			duration := float64(time.Since(start).Milliseconds())
			status := "ok"
			if err != nil {
				status = "error"
			}

			attrs := metric.WithAttributes(
				attribute.String("rpc_method", method),
				attribute.String("rpc_status", status),
			)

			i.serverDuration.Record(ctx, duration, attrs)
			i.serverRequestCount.Add(ctx, 1, attrs)

			if i.cfg.RecordPayloadSize && i.serverResponseSize != nil {
				i.serverResponseSize.Record(ctx, int64(len(resp)),
					metric.WithAttributes(attribute.String("rpc_method", method)))
			}
		}

		// Wrap error with errors.E for automatic span recording.
		if err != nil {
			return resp, errors.E(ctx, errors.CatInternal, errors.TypeUnknown, err)
		}

		return resp, nil
	}
}

// StreamServerInterceptor returns a stream server interceptor with tracing and metrics.
func (i *Interceptor) StreamServerInterceptor() interceptor.StreamServerInterceptor {
	return func(ctx context.Context, stream interceptor.ServerStream, info *interceptor.StreamServerInfo, handler interceptor.StreamHandler) error {
		method := info.Package + "/" + info.Service + "/" + info.Method
		start := time.Now()

		// Start span if tracing is enabled.
		if i.cfg.EnableTracing {
			var sp span.Span
			ctx, sp = span.New(ctx,
				span.WithName(method),
				span.WithSpanStartOption(trace.WithSpanKind(trace.SpanKindServer)),
			)
			defer sp.End()

			// Add span attributes.
			sp.Span.SetAttributes(
				attribute.String("rpc.system", "claw"),
				attribute.String("rpc.service", info.Service),
				attribute.String("rpc.method", info.Method),
				attribute.String("rpc.package", info.Package),
				attribute.Int64("rpc.session_id", int64(info.SessionID)),
				attribute.String("rpc.type", info.RPCType.String()),
			)
		}

		// Call handler.
		err := handler(ctx, stream)

		// Record metrics.
		if i.cfg.EnableMetrics {
			duration := float64(time.Since(start).Milliseconds())
			status := "ok"
			if err != nil {
				status = "error"
			}

			attrs := metric.WithAttributes(
				attribute.String("rpc_method", method),
				attribute.String("rpc_status", status),
			)

			i.serverDuration.Record(ctx, duration, attrs)
			i.serverRequestCount.Add(ctx, 1, attrs)
		}

		// Wrap error with errors.E for automatic span recording.
		if err != nil {
			return errors.E(ctx, errors.CatInternal, errors.TypeUnknown, err)
		}

		return nil
	}
}

// UnaryClientInterceptor returns a unary client interceptor with tracing and metrics.
func (i *Interceptor) UnaryClientInterceptor() interceptor.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req []byte, invoker interceptor.UnaryInvoker) ([]byte, error) {
		start := time.Now()

		// Start span if tracing is enabled.
		if i.cfg.EnableTracing {
			var sp span.Span
			ctx, sp = span.New(ctx,
				span.WithName(method),
				span.WithSpanStartOption(trace.WithSpanKind(trace.SpanKindClient)),
			)
			defer sp.End()

			// Add span attributes.
			sp.Span.SetAttributes(
				attribute.String("rpc.system", "claw"),
				attribute.String("rpc.method", method),
			)

			if i.cfg.RecordPayloadSize {
				sp.Span.SetAttributes(attribute.Int("rpc.request.size", len(req)))
			}
		}

		// Record request size.
		if i.cfg.EnableMetrics && i.cfg.RecordPayloadSize && i.clientRequestSize != nil {
			i.clientRequestSize.Record(ctx, int64(len(req)),
				metric.WithAttributes(attribute.String("rpc_method", method)))
		}

		// Call invoker.
		resp, err := invoker(ctx, req)

		// Record metrics.
		if i.cfg.EnableMetrics {
			duration := float64(time.Since(start).Milliseconds())
			status := "ok"
			if err != nil {
				status = "error"
			}

			attrs := metric.WithAttributes(
				attribute.String("rpc_method", method),
				attribute.String("rpc_status", status),
			)

			i.clientDuration.Record(ctx, duration, attrs)
			i.clientRequestCount.Add(ctx, 1, attrs)

			if i.cfg.RecordPayloadSize && i.clientResponseSize != nil {
				i.clientResponseSize.Record(ctx, int64(len(resp)),
					metric.WithAttributes(attribute.String("rpc_method", method)))
			}
		}

		// Wrap error with errors.E for automatic span recording.
		if err != nil {
			return resp, errors.E(ctx, errors.CatInternal, errors.TypeConn, err)
		}

		return resp, nil
	}
}

// StreamClientInterceptor returns a stream client interceptor with tracing and metrics.
func (i *Interceptor) StreamClientInterceptor() interceptor.StreamClientInterceptor {
	return func(ctx context.Context, method string, streamer interceptor.ClientStreamer) (interceptor.ClientStream, error) {
		start := time.Now()

		// Start span if tracing is enabled.
		if i.cfg.EnableTracing {
			var sp span.Span
			ctx, sp = span.New(ctx,
				span.WithName(method),
				span.WithSpanStartOption(trace.WithSpanKind(trace.SpanKindClient)),
			)
			defer sp.End()

			// Add span attributes.
			sp.Span.SetAttributes(
				attribute.String("rpc.system", "claw"),
				attribute.String("rpc.method", method),
			)
		}

		// Create the actual stream.
		stream, err := streamer(ctx)

		// Record metrics for stream creation.
		if i.cfg.EnableMetrics {
			duration := float64(time.Since(start).Milliseconds())
			status := "ok"
			if err != nil {
				status = "error"
			}

			attrs := metric.WithAttributes(
				attribute.String("rpc_method", method),
				attribute.String("rpc_status", status),
			)

			i.clientDuration.Record(ctx, duration, attrs)
			i.clientRequestCount.Add(ctx, 1, attrs)
		}

		if err != nil {
			return nil, errors.E(ctx, errors.CatInternal, errors.TypeConn, err)
		}

		return stream, nil
	}
}

// NewServerInterceptors creates server interceptors from a Config.
func NewServerInterceptors(ctx context.Context, cfg Config) (interceptor.UnaryServerInterceptor, interceptor.StreamServerInterceptor, error) {
	i, err := New(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	return i.UnaryServerInterceptor(), i.StreamServerInterceptor(), nil
}

// NewClientInterceptors creates client interceptors from a Config.
func NewClientInterceptors(ctx context.Context, cfg Config) (interceptor.UnaryClientInterceptor, interceptor.StreamClientInterceptor, error) {
	i, err := New(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	return i.UnaryClientInterceptor(), i.StreamClientInterceptor(), nil
}

// UnaryServerInterceptor returns a default unary server interceptor.
func UnaryServerInterceptor(ctx context.Context) (interceptor.UnaryServerInterceptor, error) {
	i, err := New(ctx, DefaultConfig())
	if err != nil {
		return nil, err
	}
	return i.UnaryServerInterceptor(), nil
}

// StreamServerInterceptor returns a default stream server interceptor.
func StreamServerInterceptor(ctx context.Context) (interceptor.StreamServerInterceptor, error) {
	i, err := New(ctx, DefaultConfig())
	if err != nil {
		return nil, err
	}
	return i.StreamServerInterceptor(), nil
}

// UnaryClientInterceptor returns a default unary client interceptor.
func UnaryClientInterceptor(ctx context.Context) (interceptor.UnaryClientInterceptor, error) {
	i, err := New(ctx, DefaultConfig())
	if err != nil {
		return nil, err
	}
	return i.UnaryClientInterceptor(), nil
}

// StreamClientInterceptor returns a default stream client interceptor.
func StreamClientInterceptor(ctx context.Context) (interceptor.StreamClientInterceptor, error) {
	i, err := New(ctx, DefaultConfig())
	if err != nil {
		return nil, err
	}
	return i.StreamClientInterceptor(), nil
}
