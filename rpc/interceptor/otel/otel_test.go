package otel

import (
	"errors"
	"iter"
	"testing"

	"github.com/gostdlib/base/context"
	"github.com/kylelemons/godebug/pretty"

	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.EnableTracing {
		t.Errorf("[TestDefaultConfig]: EnableTracing = false, want true")
	}
	if !cfg.EnableMetrics {
		t.Errorf("[TestDefaultConfig]: EnableMetrics = false, want true")
	}
	if !cfg.RecordPayloadSize {
		t.Errorf("[TestDefaultConfig]: RecordPayloadSize = false, want true")
	}
	if cfg.MeterProvider != nil {
		t.Errorf("[TestDefaultConfig]: MeterProvider = %v, want nil", cfg.MeterProvider)
	}
	if cfg.TraceRules != nil {
		t.Errorf("[TestDefaultConfig]: TraceRules = %v, want nil", cfg.TraceRules)
	}
}

func TestTraceRulesCompile(t *testing.T) {
	tests := []struct {
		name    string
		rules   *TraceRules
		wantErr bool
	}{
		{
			name:    "Success: nil rules",
			rules:   nil,
			wantErr: false,
		},
		{
			name: "Success: valid CIDR ranges",
			rules: &TraceRules{
				IPRanges: []string{"10.0.0.0/8", "192.168.1.0/24", "172.16.0.0/12"},
			},
			wantErr: false,
		},
		{
			name: "Success: empty rules",
			rules: &TraceRules{
				IPRanges: []string{},
				Methods:  []string{},
				Metadata: map[string]string{},
			},
			wantErr: false,
		},
		{
			name: "Error: invalid CIDR",
			rules: &TraceRules{
				IPRanges: []string{"invalid-cidr"},
			},
			wantErr: true,
		},
		{
			name: "Error: invalid IP in CIDR",
			rules: &TraceRules{
				IPRanges: []string{"999.999.999.999/8"},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := test.rules.compile()
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestTraceRulesCompile](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestTraceRulesCompile](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		// Verify CIDRs were compiled
		if test.rules != nil && len(test.rules.IPRanges) > 0 {
			if len(test.rules.cidrs) != len(test.rules.IPRanges) {
				t.Errorf("[TestTraceRulesCompile](%s): compiled %d CIDRs, want %d",
					test.name, len(test.rules.cidrs), len(test.rules.IPRanges))
			}
		}
	}
}

func TestTraceRulesMatchesIP(t *testing.T) {
	rules := &TraceRules{
		IPRanges: []string{"10.0.0.0/8", "192.168.1.0/24"},
	}
	if err := rules.compile(); err != nil {
		t.Fatalf("[TestTraceRulesMatchesIP]: compile error: %v", err)
	}

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{
			name: "Success: matches first range",
			ip:   "10.1.2.3",
			want: true,
		},
		{
			name: "Success: matches second range",
			ip:   "192.168.1.100",
			want: true,
		},
		{
			name: "Success: no match",
			ip:   "8.8.8.8",
			want: false,
		},
		{
			name: "Success: invalid IP returns false",
			ip:   "invalid-ip",
			want: false,
		},
		{
			name: "Success: empty IP returns false",
			ip:   "",
			want: false,
		},
		{
			name: "Success: edge of range",
			ip:   "10.255.255.255",
			want: true,
		},
		{
			name: "Success: just outside range",
			ip:   "192.168.2.1",
			want: false,
		},
	}

	for _, test := range tests {
		got := rules.matchesIP(test.ip)
		if got != test.want {
			t.Errorf("[TestTraceRulesMatchesIP](%s): got %v, want %v", test.name, got, test.want)
		}
	}
}

func TestTraceRulesMatchesIPNilRules(t *testing.T) {
	var rules *TraceRules
	if rules.matchesIP("10.0.0.1") {
		t.Error("[TestTraceRulesMatchesIPNilRules]: nil rules should return false")
	}
}

func TestTraceRulesMatchesMethod(t *testing.T) {
	rules := &TraceRules{
		Methods: []string{"auth/Login", "payment/Charge", "GetStatus"},
	}

	tests := []struct {
		name   string
		method string
		want   bool
	}{
		{
			name:   "Success: exact match",
			method: "auth/Login",
			want:   true,
		},
		{
			name:   "Success: suffix match",
			method: "pkg/auth/Login",
			want:   true,
		},
		{
			name:   "Success: simple method match",
			method: "service/GetStatus",
			want:   true,
		},
		{
			name:   "Success: no match",
			method: "other/Method",
			want:   false,
		},
		{
			name:   "Success: partial no match",
			method: "authLogin",
			want:   false,
		},
	}

	for _, test := range tests {
		got := rules.matchesMethod(test.method)
		if got != test.want {
			t.Errorf("[TestTraceRulesMatchesMethod](%s): got %v, want %v", test.name, got, test.want)
		}
	}
}

func TestTraceRulesMatchesMethodNilRules(t *testing.T) {
	var rules *TraceRules
	if rules.matchesMethod("auth/Login") {
		t.Error("[TestTraceRulesMatchesMethodNilRules]: nil rules should return false")
	}
}

func TestTraceRulesMatchesMetadata(t *testing.T) {
	ctx := t.Context()
	rules := &TraceRules{
		Metadata: map[string]string{
			"x-vip-client": "true",
			"x-debug":      "*",
		},
	}

	tests := []struct {
		name     string
		metadata []msgs.Metadata
		want     bool
	}{
		{
			name: "Success: exact match",
			metadata: func() []msgs.Metadata {
				md := msgs.NewMetadata(ctx)
				md.SetKey("x-vip-client").SetValue([]byte("true"))
				return []msgs.Metadata{md}
			}(),
			want: true,
		},
		{
			name: "Success: wildcard match",
			metadata: func() []msgs.Metadata {
				md := msgs.NewMetadata(ctx)
				md.SetKey("x-debug").SetValue([]byte("anything"))
				return []msgs.Metadata{md}
			}(),
			want: true,
		},
		{
			name: "Success: no match wrong key",
			metadata: func() []msgs.Metadata {
				md := msgs.NewMetadata(ctx)
				md.SetKey("x-other").SetValue([]byte("value"))
				return []msgs.Metadata{md}
			}(),
			want: false,
		},
		{
			name: "Success: no match wrong value",
			metadata: func() []msgs.Metadata {
				md := msgs.NewMetadata(ctx)
				md.SetKey("x-vip-client").SetValue([]byte("false"))
				return []msgs.Metadata{md}
			}(),
			want: false,
		},
		{
			name:     "Success: empty metadata",
			metadata: []msgs.Metadata{},
			want:     false,
		},
		{
			name:     "Success: nil metadata",
			metadata: nil,
			want:     false,
		},
	}

	for _, test := range tests {
		got := rules.matchesMetadata(test.metadata)
		if got != test.want {
			t.Errorf("[TestTraceRulesMatchesMetadata](%s): got %v, want %v", test.name, got, test.want)
		}
		// Release metadata to avoid leaks
		for _, md := range test.metadata {
			md.Release(ctx)
		}
	}
}

func TestTraceRulesMatchesMetadataNilRules(t *testing.T) {
	var rules *TraceRules
	if rules.matchesMetadata(nil) {
		t.Error("[TestTraceRulesMatchesMetadataNilRules]: nil rules should return false")
	}
}

func TestTraceRulesShouldTrace(t *testing.T) {
	ctx := t.Context()
	rules := &TraceRules{
		IPRanges: []string{"10.0.0.0/8"},
		Methods:  []string{"auth/Login"},
		Metadata: map[string]string{"x-debug": "*"},
	}
	if err := rules.compile(); err != nil {
		t.Fatalf("[TestTraceRulesShouldTrace]: compile error: %v", err)
	}

	tests := []struct {
		name     string
		ip       string
		method   string
		metadata []msgs.Metadata
		want     bool
	}{
		{
			name:   "Success: IP match",
			ip:     "10.1.2.3",
			method: "other/Method",
			want:   true,
		},
		{
			name:   "Success: method match",
			ip:     "8.8.8.8",
			method: "auth/Login",
			want:   true,
		},
		{
			name:   "Success: metadata match",
			ip:     "8.8.8.8",
			method: "other/Method",
			metadata: func() []msgs.Metadata {
				md := msgs.NewMetadata(ctx)
				md.SetKey("x-debug").SetValue([]byte("1"))
				return []msgs.Metadata{md}
			}(),
			want: true,
		},
		{
			name:   "Success: no match",
			ip:     "8.8.8.8",
			method: "other/Method",
			want:   false,
		},
	}

	for _, test := range tests {
		got := rules.ShouldTrace(test.ip, test.method, test.metadata)
		if got != test.want {
			t.Errorf("[TestTraceRulesShouldTrace](%s): got %v, want %v", test.name, got, test.want)
		}
		for _, md := range test.metadata {
			md.Release(ctx)
		}
	}
}

func TestTraceRulesShouldTraceNilRules(t *testing.T) {
	var rules *TraceRules
	if rules.ShouldTrace("10.0.0.1", "auth/Login", nil) {
		t.Error("[TestTraceRulesShouldTraceNilRules]: nil rules should return false")
	}
}

func TestNew(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "Success: default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "Success: metrics disabled",
			cfg: Config{
				EnableTracing: true,
				EnableMetrics: false,
			},
			wantErr: false,
		},
		{
			name: "Success: tracing disabled",
			cfg: Config{
				EnableTracing: false,
				EnableMetrics: true,
			},
			wantErr: false,
		},
		{
			name: "Success: with valid trace rules",
			cfg: Config{
				EnableTracing: true,
				EnableMetrics: true,
				TraceRules: &TraceRules{
					IPRanges: []string{"10.0.0.0/8"},
					Methods:  []string{"auth/Login"},
				},
			},
			wantErr: false,
		},
		{
			name: "Error: invalid trace rules",
			cfg: Config{
				TraceRules: &TraceRules{
					IPRanges: []string{"invalid-cidr"},
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		i, err := New(ctx, test.cfg)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestNew](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestNew](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if i == nil {
			t.Errorf("[TestNew](%s): got nil interceptor, want non-nil", test.name)
		}
	}
}

func TestNewServerInterceptors(t *testing.T) {
	ctx := t.Context()

	unary, stream, err := NewServerInterceptors(ctx, DefaultConfig())
	if err != nil {
		t.Fatalf("[TestNewServerInterceptors]: got err = %v, want nil", err)
	}
	if unary == nil {
		t.Error("[TestNewServerInterceptors]: unary interceptor is nil")
	}
	if stream == nil {
		t.Error("[TestNewServerInterceptors]: stream interceptor is nil")
	}
}

func TestNewClientInterceptors(t *testing.T) {
	ctx := t.Context()

	unary, stream, err := NewClientInterceptors(ctx, DefaultConfig())
	if err != nil {
		t.Fatalf("[TestNewClientInterceptors]: got err = %v, want nil", err)
	}
	if unary == nil {
		t.Error("[TestNewClientInterceptors]: unary interceptor is nil")
	}
	if stream == nil {
		t.Error("[TestNewClientInterceptors]: stream interceptor is nil")
	}
}

func TestUnaryServerInterceptor(t *testing.T) {
	ctx := t.Context()

	interceptorFunc, err := UnaryServerInterceptor(ctx)
	if err != nil {
		t.Fatalf("[TestUnaryServerInterceptor]: got err = %v, want nil", err)
	}

	// Test that the interceptor calls the handler
	called := false
	handler := func(ctx context.Context, req []byte) ([]byte, error) {
		called = true
		return []byte("response"), nil
	}

	info := &interceptor.UnaryServerInfo{
		Package:   "test",
		Service:   "TestService",
		Method:    "TestMethod",
		SessionID: 1,
	}

	resp, err := interceptorFunc(ctx, []byte("request"), info, handler)
	if err != nil {
		t.Errorf("[TestUnaryServerInterceptor]: got err = %v, want nil", err)
	}
	if !called {
		t.Error("[TestUnaryServerInterceptor]: handler was not called")
	}
	if diff := pretty.Compare(resp, []byte("response")); diff != "" {
		t.Errorf("[TestUnaryServerInterceptor]: response diff (-got +want):\n%s", diff)
	}
}

func TestUnaryServerInterceptorError(t *testing.T) {
	ctx := t.Context()

	interceptorFunc, err := UnaryServerInterceptor(ctx)
	if err != nil {
		t.Fatalf("[TestUnaryServerInterceptorError]: got err = %v, want nil", err)
	}

	// Test that the interceptor wraps errors
	testErr := errors.New("test error")
	handler := func(ctx context.Context, req []byte) ([]byte, error) {
		return nil, testErr
	}

	info := &interceptor.UnaryServerInfo{
		Package:   "test",
		Service:   "TestService",
		Method:    "TestMethod",
		SessionID: 1,
	}

	_, err = interceptorFunc(ctx, []byte("request"), info, handler)
	if err == nil {
		t.Error("[TestUnaryServerInterceptorError]: got err == nil, want err != nil")
	}
}

// fakeServerStream implements interceptor.ServerStream for testing.
type fakeServerStream struct {
	ctx context.Context
}

func (f *fakeServerStream) Send(payload []byte) error {
	return nil
}

func (f *fakeServerStream) Recv() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {}
}

func (f *fakeServerStream) Context() context.Context {
	return f.ctx
}

func TestStreamServerInterceptor(t *testing.T) {
	ctx := t.Context()

	interceptorFunc, err := StreamServerInterceptor(ctx)
	if err != nil {
		t.Fatalf("[TestStreamServerInterceptor]: got err = %v, want nil", err)
	}

	// Test that the interceptor calls the handler
	called := false
	handler := func(ctx context.Context, stream interceptor.ServerStream) error {
		called = true
		return nil
	}

	info := &interceptor.StreamServerInfo{
		Package:   "test",
		Service:   "TestService",
		Method:    "TestMethod",
		SessionID: 1,
	}

	stream := &fakeServerStream{ctx: ctx}
	err = interceptorFunc(ctx, stream, info, handler)
	if err != nil {
		t.Errorf("[TestStreamServerInterceptor]: got err = %v, want nil", err)
	}
	if !called {
		t.Error("[TestStreamServerInterceptor]: handler was not called")
	}
}

func TestStreamServerInterceptorError(t *testing.T) {
	ctx := t.Context()

	interceptorFunc, err := StreamServerInterceptor(ctx)
	if err != nil {
		t.Fatalf("[TestStreamServerInterceptorError]: got err = %v, want nil", err)
	}

	testErr := errors.New("test error")
	handler := func(ctx context.Context, stream interceptor.ServerStream) error {
		return testErr
	}

	info := &interceptor.StreamServerInfo{
		Package:   "test",
		Service:   "TestService",
		Method:    "TestMethod",
		SessionID: 1,
	}

	stream := &fakeServerStream{ctx: ctx}
	err = interceptorFunc(ctx, stream, info, handler)
	if err == nil {
		t.Error("[TestStreamServerInterceptorError]: got err == nil, want err != nil")
	}
}

func TestUnaryClientInterceptor(t *testing.T) {
	ctx := t.Context()

	interceptorFunc, err := UnaryClientInterceptor(ctx)
	if err != nil {
		t.Fatalf("[TestUnaryClientInterceptor]: got err = %v, want nil", err)
	}

	// Test that the interceptor calls the invoker
	called := false
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		called = true
		return []byte("response"), nil
	}

	resp, err := interceptorFunc(ctx, "test/Service/Method", []byte("request"), invoker)
	if err != nil {
		t.Errorf("[TestUnaryClientInterceptor]: got err = %v, want nil", err)
	}
	if !called {
		t.Error("[TestUnaryClientInterceptor]: invoker was not called")
	}
	if diff := pretty.Compare(resp, []byte("response")); diff != "" {
		t.Errorf("[TestUnaryClientInterceptor]: response diff (-got +want):\n%s", diff)
	}
}

func TestUnaryClientInterceptorError(t *testing.T) {
	ctx := t.Context()

	interceptorFunc, err := UnaryClientInterceptor(ctx)
	if err != nil {
		t.Fatalf("[TestUnaryClientInterceptorError]: got err = %v, want nil", err)
	}

	testErr := errors.New("test error")
	invoker := func(ctx context.Context, req []byte) ([]byte, error) {
		return nil, testErr
	}

	_, err = interceptorFunc(ctx, "test/Service/Method", []byte("request"), invoker)
	if err == nil {
		t.Error("[TestUnaryClientInterceptorError]: got err == nil, want err != nil")
	}
}

// fakeClientStream implements interceptor.ClientStream for testing.
type fakeClientStream struct{}

func (f *fakeClientStream) Send(ctx context.Context, payload []byte) error {
	return nil
}

func (f *fakeClientStream) Recv(ctx context.Context) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {}
}

func (f *fakeClientStream) CloseSend() error {
	return nil
}

func (f *fakeClientStream) Close() error {
	return nil
}

func (f *fakeClientStream) Err() error {
	return nil
}

func TestStreamClientInterceptor(t *testing.T) {
	ctx := t.Context()

	interceptorFunc, err := StreamClientInterceptor(ctx)
	if err != nil {
		t.Fatalf("[TestStreamClientInterceptor]: got err = %v, want nil", err)
	}

	// Test that the interceptor calls the streamer
	called := false
	streamer := func(ctx context.Context) (interceptor.ClientStream, error) {
		called = true
		return &fakeClientStream{}, nil
	}

	stream, err := interceptorFunc(ctx, "test/Service/Method", streamer)
	if err != nil {
		t.Errorf("[TestStreamClientInterceptor]: got err = %v, want nil", err)
	}
	if !called {
		t.Error("[TestStreamClientInterceptor]: streamer was not called")
	}
	if stream == nil {
		t.Error("[TestStreamClientInterceptor]: got nil stream, want non-nil")
	}
}

func TestStreamClientInterceptorError(t *testing.T) {
	ctx := t.Context()

	interceptorFunc, err := StreamClientInterceptor(ctx)
	if err != nil {
		t.Fatalf("[TestStreamClientInterceptorError]: got err = %v, want nil", err)
	}

	testErr := errors.New("test error")
	streamer := func(ctx context.Context) (interceptor.ClientStream, error) {
		return nil, testErr
	}

	_, err = interceptorFunc(ctx, "test/Service/Method", streamer)
	if err == nil {
		t.Error("[TestStreamClientInterceptorError]: got err == nil, want err != nil")
	}
}

func TestInterceptorWithMetricsDisabled(t *testing.T) {
	ctx := t.Context()

	cfg := Config{
		EnableTracing: true,
		EnableMetrics: false,
	}

	i, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("[TestInterceptorWithMetricsDisabled]: got err = %v, want nil", err)
	}

	// Verify metrics instruments are nil when metrics disabled
	if i.serverDuration != nil {
		t.Error("[TestInterceptorWithMetricsDisabled]: serverDuration should be nil")
	}
	if i.clientDuration != nil {
		t.Error("[TestInterceptorWithMetricsDisabled]: clientDuration should be nil")
	}
}

func TestInterceptorWithPayloadSizeDisabled(t *testing.T) {
	ctx := t.Context()

	cfg := Config{
		EnableTracing:     true,
		EnableMetrics:     true,
		RecordPayloadSize: false,
	}

	i, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("[TestInterceptorWithPayloadSizeDisabled]: got err = %v, want nil", err)
	}

	// Verify size metrics instruments are nil when payload size recording disabled
	if i.serverRequestSize != nil {
		t.Error("[TestInterceptorWithPayloadSizeDisabled]: serverRequestSize should be nil")
	}
	if i.serverResponseSize != nil {
		t.Error("[TestInterceptorWithPayloadSizeDisabled]: serverResponseSize should be nil")
	}
	if i.clientRequestSize != nil {
		t.Error("[TestInterceptorWithPayloadSizeDisabled]: clientRequestSize should be nil")
	}
	if i.clientResponseSize != nil {
		t.Error("[TestInterceptorWithPayloadSizeDisabled]: clientResponseSize should be nil")
	}
}
