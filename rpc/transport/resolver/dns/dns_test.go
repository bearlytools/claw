package dns

import (
	"testing"

	"github.com/bearlytools/claw/rpc/transport/resolver"
)

func TestBuilder(t *testing.T) {
	b := &builder{}
	if b.Scheme() != "dns" {
		t.Errorf("[TestBuilder]: Scheme() = %q, want %q", b.Scheme(), "dns")
	}
}

func TestDNSResolverWithIP(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name     string
		endpoint string
		wantAddr string
	}{
		{
			name:     "Success: IPv4 with port",
			endpoint: "192.168.1.1:8080",
			wantAddr: "192.168.1.1:8080",
		},
		{
			name:     "Success: IPv4 without port uses default",
			endpoint: "192.168.1.1",
			wantAddr: "192.168.1.1:443",
		},
		{
			name:     "Success: IPv6 with port",
			endpoint: "[::1]:8080",
			wantAddr: "[::1]:8080",
		},
	}

	b := &builder{}
	for _, test := range tests {
		target := resolver.Target{
			Scheme:   "dns",
			Endpoint: test.endpoint,
		}

		r, err := b.Build(target, resolver.BuildOptions{})
		if err != nil {
			t.Errorf("[TestDNSResolverWithIP](%s): Build() error: %v", test.name, err)
			continue
		}
		defer r.Close()

		got, err := r.Resolve(ctx)
		if err != nil {
			t.Errorf("[TestDNSResolverWithIP](%s): Resolve() error: %v", test.name, err)
			continue
		}

		if len(got) != 1 {
			t.Errorf("[TestDNSResolverWithIP](%s): got %d addresses, want 1", test.name, len(got))
			continue
		}

		if got[0].Addr != test.wantAddr {
			t.Errorf("[TestDNSResolverWithIP](%s): got addr %q, want %q", test.name, got[0].Addr, test.wantAddr)
		}
	}
}

func TestDNSResolverLocalhost(t *testing.T) {
	ctx := t.Context()

	target := resolver.Target{
		Scheme:   "dns",
		Endpoint: "localhost:8080",
	}

	b := &builder{}
	r, err := b.Build(target, resolver.BuildOptions{})
	if err != nil {
		t.Fatalf("[TestDNSResolverLocalhost]: Build() error: %v", err)
	}
	defer r.Close()

	got, err := r.Resolve(ctx)
	if err != nil {
		t.Fatalf("[TestDNSResolverLocalhost]: Resolve() error: %v", err)
	}

	if len(got) == 0 {
		t.Error("[TestDNSResolverLocalhost]: got 0 addresses, want at least 1")
		return
	}

	// Localhost should resolve to 127.0.0.1 or ::1
	found := false
	for _, addr := range got {
		if addr.Addr == "127.0.0.1:8080" || addr.Addr == "[::1]:8080" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("[TestDNSResolverLocalhost]: expected localhost address, got %v", got)
	}
}

func TestDNSResolverClose(t *testing.T) {
	b := &builder{}
	r, err := b.Build(resolver.Target{Endpoint: "localhost:8080"}, resolver.BuildOptions{})
	if err != nil {
		t.Fatalf("[TestDNSResolverClose]: Build() error: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Errorf("[TestDNSResolverClose]: Close() error: %v", err)
	}
}

func TestDNSRegistered(t *testing.T) {
	// The DNS resolver should be auto-registered via init()
	b, ok := resolver.Get("dns")
	if !ok {
		t.Error("[TestDNSRegistered]: dns resolver not registered")
		return
	}

	if b.Scheme() != "dns" {
		t.Errorf("[TestDNSRegistered]: Scheme() = %q, want %q", b.Scheme(), "dns")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.defaultPort != "443" {
		t.Errorf("[TestDefaultConfig]: defaultPort = %q, want %q", cfg.defaultPort, "443")
	}
	if cfg.useSRV {
		t.Error("[TestDefaultConfig]: useSRV should be false by default")
	}
}

func TestWithDefaultPort(t *testing.T) {
	cfg := defaultConfig()
	WithDefaultPort("8080")(cfg)
	if cfg.defaultPort != "8080" {
		t.Errorf("[TestWithDefaultPort]: defaultPort = %q, want %q", cfg.defaultPort, "8080")
	}
}

func TestWithSRV(t *testing.T) {
	cfg := defaultConfig()
	WithSRV("grpc", "tcp")(cfg)
	if !cfg.useSRV {
		t.Error("[TestWithSRV]: useSRV should be true")
	}
	if cfg.srvService != "grpc" {
		t.Errorf("[TestWithSRV]: srvService = %q, want %q", cfg.srvService, "grpc")
	}
	if cfg.srvProto != "tcp" {
		t.Errorf("[TestWithSRV]: srvProto = %q, want %q", cfg.srvProto, "tcp")
	}
}
