// Package passthrough provides a resolver that passes through addresses unchanged.
// This is the default resolver used when no scheme is specified.
package passthrough

import (
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/transport/resolver"
)

func init() {
	resolver.Register(&builder{})
}

type builder struct{}

func (b *builder) Scheme() string {
	return "passthrough"
}

func (b *builder) Build(target resolver.Target, opts resolver.BuildOptions) (resolver.Resolver, error) {
	return &passthroughResolver{addr: target.Endpoint}, nil
}

type passthroughResolver struct {
	addr string
}

func (r *passthroughResolver) Resolve(ctx context.Context) ([]resolver.Address, error) {
	return []resolver.Address{{Addr: r.addr}}, nil
}

func (r *passthroughResolver) Close() error {
	return nil
}
