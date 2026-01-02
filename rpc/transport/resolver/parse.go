package resolver

import (
	"errors"
	"strings"
)

// DefaultScheme is used when no scheme is specified in the target.
const DefaultScheme = "passthrough"

// Parse parses a target string into its components.
// Format: scheme://authority/endpoint
//
// Examples:
//   - "dns:///myservice.namespace:8080" -> {dns, "", myservice.namespace:8080}
//   - "dns://dns-server:53/myservice:8080" -> {dns, dns-server:53, myservice:8080}
//   - "passthrough:///localhost:8080" -> {passthrough, "", localhost:8080}
//   - "localhost:8080" -> {passthrough, "", localhost:8080} (bare address)
//   - "/var/run/app.sock" -> {passthrough, "", /var/run/app.sock} (unix path)
func Parse(target string) (Target, error) {
	if target == "" {
		return Target{}, errors.New("empty target")
	}

	// Handle bare addresses (no scheme)
	if !strings.Contains(target, "://") {
		return Target{
			Scheme:   DefaultScheme,
			Endpoint: target,
		}, nil
	}

	// Parse scheme://rest
	idx := strings.Index(target, "://")
	scheme := strings.ToLower(target[:idx])
	rest := target[idx+3:]

	if scheme == "" {
		return Target{}, errors.New("empty scheme")
	}

	// Parse authority and endpoint from rest
	// Format: authority/endpoint or /endpoint (empty authority)
	var authority, endpoint string

	if strings.HasPrefix(rest, "/") {
		// Empty authority: scheme:///endpoint
		endpoint = rest[1:]
	} else {
		// Has authority: scheme://authority/endpoint
		slashIdx := strings.Index(rest, "/")
		if slashIdx == -1 {
			// No endpoint, treat rest as authority
			// This handles "dns://server" without endpoint
			return Target{}, errors.New("missing endpoint after authority")
		}
		authority = rest[:slashIdx]
		endpoint = rest[slashIdx+1:]
	}

	if endpoint == "" {
		return Target{}, errors.New("empty endpoint")
	}

	return Target{
		Scheme:    scheme,
		Authority: authority,
		Endpoint:  endpoint,
	}, nil
}

// String returns the target as a formatted string.
func (t Target) String() string {
	if t.Authority == "" {
		return t.Scheme + ":///" + t.Endpoint
	}
	return t.Scheme + "://" + t.Authority + "/" + t.Endpoint
}
