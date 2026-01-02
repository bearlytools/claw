package passthrough

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"

	"github.com/bearlytools/claw/rpc/transport/resolver"
)

func TestBuilder(t *testing.T) {
	b := &builder{}
	if b.Scheme() != "passthrough" {
		t.Errorf("[TestBuilder]: Scheme() = %q, want %q", b.Scheme(), "passthrough")
	}
}

func TestPassthroughResolver(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name     string
		endpoint string
		want     []resolver.Address
	}{
		{
			name:     "Success: host and port",
			endpoint: "localhost:8080",
			want:     []resolver.Address{{Addr: "localhost:8080"}},
		},
		{
			name:     "Success: unix socket path",
			endpoint: "/var/run/app.sock",
			want:     []resolver.Address{{Addr: "/var/run/app.sock"}},
		},
		{
			name:     "Success: IP address",
			endpoint: "192.168.1.1:8080",
			want:     []resolver.Address{{Addr: "192.168.1.1:8080"}},
		},
	}

	b := &builder{}
	for _, test := range tests {
		target := resolver.Target{
			Scheme:   "passthrough",
			Endpoint: test.endpoint,
		}

		r, err := b.Build(target, resolver.BuildOptions{})
		if err != nil {
			t.Errorf("[TestPassthroughResolver](%s): Build() error: %v", test.name, err)
			continue
		}
		defer r.Close()

		got, err := r.Resolve(ctx)
		if err != nil {
			t.Errorf("[TestPassthroughResolver](%s): Resolve() error: %v", test.name, err)
			continue
		}

		if diff := pretty.Compare(got, test.want); diff != "" {
			t.Errorf("[TestPassthroughResolver](%s): diff (-got +want):\n%s", test.name, diff)
		}
	}
}

func TestPassthroughResolverClose(t *testing.T) {
	b := &builder{}
	r, err := b.Build(resolver.Target{Endpoint: "localhost:8080"}, resolver.BuildOptions{})
	if err != nil {
		t.Fatalf("[TestPassthroughResolverClose]: Build() error: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Errorf("[TestPassthroughResolverClose]: Close() error: %v", err)
	}
}

func TestPassthroughRegistered(t *testing.T) {
	// The passthrough resolver should be auto-registered via init()
	b, ok := resolver.Get("passthrough")
	if !ok {
		t.Error("[TestPassthroughRegistered]: passthrough resolver not registered")
		return
	}

	if b.Scheme() != "passthrough" {
		t.Errorf("[TestPassthroughRegistered]: Scheme() = %q, want %q", b.Scheme(), "passthrough")
	}
}
