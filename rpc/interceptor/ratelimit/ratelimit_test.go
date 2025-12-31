package ratelimit

import (
	"errors"
	"testing"
	"time"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		wantRate   float64
		wantBurst  int
	}{
		{
			name:      "Success: default values",
			cfg:       Config{},
			wantRate:  100,
			wantBurst: 10,
		},
		{
			name: "Success: custom values",
			cfg: Config{
				Rate:  50,
				Burst: 5,
			},
			wantRate:  50,
			wantBurst: 5,
		},
		{
			name: "Success: zero rate uses default",
			cfg: Config{
				Rate:  0,
				Burst: 5,
			},
			wantRate:  100,
			wantBurst: 5,
		},
		{
			name: "Success: zero burst uses default",
			cfg: Config{
				Rate:  50,
				Burst: 0,
			},
			wantRate:  50,
			wantBurst: 10,
		},
	}

	for _, test := range tests {
		l := New(test.cfg)
		if l.rate != test.wantRate {
			t.Errorf("[TestNew](%s): rate = %f, want %f", test.name, l.rate, test.wantRate)
		}
		if l.burst != test.wantBurst {
			t.Errorf("[TestNew](%s): burst = %d, want %d", test.name, l.burst, test.wantBurst)
		}
	}
}

func TestLimiterAllow(t *testing.T) {
	l := New(Config{
		Rate:  10,  // 10 requests per second
		Burst: 2,   // burst of 2
	})

	// Should allow burst requests immediately.
	if !l.allow("key1") {
		t.Error("[TestLimiterAllow]: first request should be allowed")
	}
	if !l.allow("key1") {
		t.Error("[TestLimiterAllow]: second request (within burst) should be allowed")
	}

	// Third request should be denied (burst exhausted).
	if l.allow("key1") {
		t.Error("[TestLimiterAllow]: third request should be denied (burst exhausted)")
	}

	// Different key should have its own bucket.
	if !l.allow("key2") {
		t.Error("[TestLimiterAllow]: different key should be allowed")
	}
}

func TestLimiterTokenRefill(t *testing.T) {
	l := New(Config{
		Rate:  1000, // 1000 requests per second (1 per ms)
		Burst: 1,    // burst of 1
	})

	// Use up the burst.
	l.allow("key1")
	if l.allow("key1") {
		t.Error("[TestLimiterTokenRefill]: should be denied after burst")
	}

	// Wait for a token to refill.
	time.Sleep(2 * time.Millisecond)

	// Should be allowed now.
	if !l.allow("key1") {
		t.Error("[TestLimiterTokenRefill]: should be allowed after token refill")
	}
}

func TestByMethod(t *testing.T) {
	keyFunc := ByMethod()

	tests := []struct {
		name string
		info interface{}
		want string
	}{
		{
			name: "Success: unary server info",
			info: &interceptor.UnaryServerInfo{
				Package: "pkg",
				Service: "svc",
				Method:  "Method",
			},
			want: "pkg/svc/Method",
		},
		{
			name: "Success: stream server info",
			info: &interceptor.StreamServerInfo{
				Package: "pkg2",
				Service: "svc2",
				Method:  "StreamMethod",
			},
			want: "pkg2/svc2/StreamMethod",
		},
		{
			name: "Success: unknown type",
			info: "not an info type",
			want: "unknown",
		},
	}

	for _, test := range tests {
		got := keyFunc(test.info)
		if got != test.want {
			t.Errorf("[TestByMethod](%s): got %q, want %q", test.name, got, test.want)
		}
	}
}

func TestByClient(t *testing.T) {
	ctx := t.Context()
	md := msgs.NewMetadata(ctx).SetKey("client-id").SetValue([]byte("client123"))

	keyFunc := ByClient("client-id")

	tests := []struct {
		name string
		info interface{}
		want string
	}{
		{
			name: "Success: unary server info with metadata",
			info: &interceptor.UnaryServerInfo{
				Package:  "pkg",
				Service:  "svc",
				Method:   "Method",
				Metadata: []msgs.Metadata{md},
			},
			want: "client123",
		},
		{
			name: "Success: unary server info without matching metadata",
			info: &interceptor.UnaryServerInfo{
				Package:  "pkg",
				Service:  "svc",
				Method:   "Method",
				Metadata: []msgs.Metadata{},
			},
			want: "",
		},
	}

	for _, test := range tests {
		got := keyFunc(test.info)
		if got != test.want {
			t.Errorf("[TestByClient](%s): got %q, want %q", test.name, got, test.want)
		}
	}
}

func TestByMethodAndClient(t *testing.T) {
	ctx := t.Context()
	md := msgs.NewMetadata(ctx).SetKey("api-key").SetValue([]byte("key456"))

	keyFunc := ByMethodAndClient("api-key")

	info := &interceptor.UnaryServerInfo{
		Package:  "pkg",
		Service:  "svc",
		Method:   "Method",
		Metadata: []msgs.Metadata{md},
	}

	got := keyFunc(info)
	want := "pkg/svc/Method:key456"
	if got != want {
		t.Errorf("[TestByMethodAndClient]: got %q, want %q", got, want)
	}
}

func TestUnaryServerInterceptor(t *testing.T) {
	l := New(Config{
		Rate:    1000,
		Burst:   1,
		KeyFunc: ByMethod(),
	})

	intcpt := l.UnaryServerInterceptor()
	ctx := t.Context()
	info := &interceptor.UnaryServerInfo{
		Package: "pkg",
		Service: "svc",
		Method:  "Method",
	}

	handlerCalled := false
	handler := func(ctx2 context.Context, req []byte) ([]byte, error) {
		handlerCalled = true
		return []byte("response"), nil
	}

	// First request should succeed.
	resp, err := intcpt(ctx, []byte("req"), info, handler)
	if err != nil {
		t.Errorf("[TestUnaryServerInterceptor]: first request err = %v, want nil", err)
	}
	if string(resp) != "response" {
		t.Errorf("[TestUnaryServerInterceptor]: first request resp = %q, want %q", resp, "response")
	}
	if !handlerCalled {
		t.Error("[TestUnaryServerInterceptor]: handler should have been called")
	}

	// Second request should be rate limited.
	handlerCalled = false
	_, err = intcpt(ctx, []byte("req"), info, handler)
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("[TestUnaryServerInterceptor]: second request err = %v, want ErrRateLimited", err)
	}
	if handlerCalled {
		t.Error("[TestUnaryServerInterceptor]: handler should not have been called when rate limited")
	}
}

func TestStreamServerInterceptor(t *testing.T) {
	l := New(Config{
		Rate:    1000,
		Burst:   1,
		KeyFunc: ByMethod(),
	})

	intcpt := l.StreamServerInterceptor()
	ctx := t.Context()
	info := &interceptor.StreamServerInfo{
		Package: "pkg",
		Service: "svc",
		Method:  "Method",
	}

	handlerCalled := false
	handler := func(ctx2 context.Context, stream interceptor.ServerStream) error {
		handlerCalled = true
		return nil
	}

	// First request should succeed.
	err := intcpt(ctx, nil, info, handler)
	if err != nil {
		t.Errorf("[TestStreamServerInterceptor]: first request err = %v, want nil", err)
	}
	if !handlerCalled {
		t.Error("[TestStreamServerInterceptor]: handler should have been called")
	}

	// Second request should be rate limited.
	handlerCalled = false
	err = intcpt(ctx, nil, info, handler)
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("[TestStreamServerInterceptor]: second request err = %v, want ErrRateLimited", err)
	}
	if handlerCalled {
		t.Error("[TestStreamServerInterceptor]: handler should not have been called when rate limited")
	}
}

func TestCleanup(t *testing.T) {
	l := New(Config{
		Rate:  100,
		Burst: 10,
	})

	// Add some entries.
	l.allow("key1")
	l.allow("key2")
	l.allow("key3")

	if l.Stats() != 3 {
		t.Errorf("[TestCleanup]: initial stats = %d, want 3", l.Stats())
	}

	// Cleanup with very short maxAge should remove all entries.
	time.Sleep(10 * time.Millisecond)
	l.Cleanup(time.Millisecond)

	if l.Stats() != 0 {
		t.Errorf("[TestCleanup]: after cleanup stats = %d, want 0", l.Stats())
	}
}

func TestCleanupKeepsRecentEntries(t *testing.T) {
	l := New(Config{
		Rate:  100,
		Burst: 10,
	})

	l.allow("key1")
	time.Sleep(50 * time.Millisecond)
	l.allow("key2") // More recent

	// Cleanup with maxAge that keeps key2 but removes key1.
	l.Cleanup(30 * time.Millisecond)

	if l.Stats() != 1 {
		t.Errorf("[TestCleanupKeepsRecentEntries]: stats = %d, want 1", l.Stats())
	}
}
