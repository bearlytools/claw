// Package dns provides a DNS-based resolver with SRV record support.
package dns

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/transport/resolver"
)

func init() {
	resolver.Register(&builder{})
}

type builder struct{}

func (b *builder) Scheme() string {
	return "dns"
}

func (b *builder) Build(target resolver.Target, opts resolver.BuildOptions) (resolver.Resolver, error) {
	cfg := defaultConfig()

	r := &dnsResolver{
		target:      target,
		config:      cfg,
		netResolver: net.DefaultResolver,
	}

	// If authority is specified, use a custom DNS server
	if target.Authority != "" {
		r.netResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: opts.DialTimeout,
				}
				return d.DialContext(ctx, network, target.Authority)
			},
		}
	}

	return r, nil
}

type config struct {
	// defaultPort is used when the endpoint doesn't include a port.
	defaultPort string

	// srvService and srvProto are used for SRV record lookups.
	// Example: _grpc._tcp -> LookupSRV("grpc", "tcp", endpoint)
	srvService string
	srvProto   string

	// useSRV indicates whether to try SRV records before A/AAAA.
	useSRV bool

	// resolveTimeout is the timeout for DNS resolution.
	resolveTimeout time.Duration
}

func defaultConfig() *config {
	return &config{
		defaultPort:    "443",
		resolveTimeout: 10 * time.Second,
	}
}

type dnsResolver struct {
	target      resolver.Target
	config      *config
	netResolver *net.Resolver
}

func (r *dnsResolver) Resolve(ctx context.Context) ([]resolver.Address, error) {
	// Apply timeout if set
	if r.config.resolveTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.resolveTimeout)
		defer cancel()
	}

	// Try SRV lookup first if configured
	if r.config.useSRV && r.config.srvService != "" && r.config.srvProto != "" {
		addrs, err := r.resolveSRV(ctx)
		if err == nil && len(addrs) > 0 {
			return addrs, nil
		}
		// Fall through to A/AAAA on SRV failure
	}

	// Resolve host to IP addresses
	return r.resolveHost(ctx)
}

func (r *dnsResolver) resolveSRV(ctx context.Context) ([]resolver.Address, error) {
	_, records, err := r.netResolver.LookupSRV(ctx, r.config.srvService, r.config.srvProto, r.target.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("SRV lookup failed: %w", err)
	}

	addrs := make([]resolver.Address, 0, len(records))
	for _, srv := range records {
		// Remove trailing dot from target if present
		target := strings.TrimSuffix(srv.Target, ".")
		addr := net.JoinHostPort(target, strconv.Itoa(int(srv.Port)))
		addrs = append(addrs, resolver.Address{
			Addr:     addr,
			Priority: uint32(srv.Priority),
			Weight:   uint32(srv.Weight),
		})
	}
	return addrs, nil
}

func (r *dnsResolver) resolveHost(ctx context.Context) ([]resolver.Address, error) {
	endpoint := r.target.Endpoint

	// Split host and port
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		// No port specified, use default
		host = endpoint
		port = r.config.defaultPort
	}

	// Check if host is already an IP address
	if ip := net.ParseIP(host); ip != nil {
		return []resolver.Address{{Addr: net.JoinHostPort(host, port)}}, nil
	}

	// Lookup IP addresses
	ips, err := r.netResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed for %q: %w", host, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for %q", host)
	}

	addrs := make([]resolver.Address, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, resolver.Address{
			Addr: net.JoinHostPort(ip.String(), port),
		})
	}
	return addrs, nil
}

func (r *dnsResolver) Close() error {
	return nil
}
