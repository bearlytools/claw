// Package reflection provides a Reflection API for claw RPC servers.
//
// The reflection service allows clients to discover registered services and methods
// at runtime. It is secured through IP restrictions (CIDR ranges) and/or token
// validation, both of which must pass if configured (AND logic).
//
// # Basic Usage
//
// Enable reflection on a server with IP restrictions only:
//
//	srv := server.New()
//	srv.Register(ctx, "myapp", "UserService", "GetUser", handler)
//
//	reflectionSrv, err := reflection.Enable(ctx, srv, reflection.Config{
//	    AllowedCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16"},
//	})
//
// # Token Validation
//
// For production use, configure a TokenValidator to securely validate auth tokens.
// The validator receives the raw token from the configured header (default: "authorization").
//
// JWT validation example:
//
//	config := reflection.Config{
//	    AllowedCIDRs: []string{"10.0.0.0/8"},
//	    TokenValidator: func(ctx context.Context, token string) error {
//	        // Strip "Bearer " prefix if present
//	        token = strings.TrimPrefix(token, "Bearer ")
//
//	        // Parse and validate JWT with your signing key
//	        claims, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
//	            return []byte(os.Getenv("JWT_SECRET")), nil
//	        })
//	        if err != nil {
//	            return err
//	        }
//	        if !claims.Valid {
//	            return errors.New("invalid token")
//	        }
//	        return nil
//	    },
//	}
//
// Secret store integration example (HashiCorp Vault, AWS Secrets Manager, etc.):
//
//	config := reflection.Config{
//	    TokenValidator: func(ctx context.Context, token string) error {
//	        // Lookup valid tokens from secret store
//	        validTokens, err := secretStore.GetSecret(ctx, "reflection-tokens")
//	        if err != nil {
//	            return err
//	        }
//	        for _, valid := range validTokens {
//	            if subtle.ConstantTimeCompare([]byte(token), []byte(valid)) == 1 {
//	                return nil
//	            }
//	        }
//	        return errors.New("token not found")
//	    },
//	}
//
// HMAC verification example:
//
//	config := reflection.Config{
//	    AuthHeader: "x-signature",
//	    TokenValidator: func(ctx context.Context, signature string) error {
//	        // Verify HMAC signature
//	        key := []byte(os.Getenv("HMAC_KEY"))
//	        mac := hmac.New(sha256.New, key)
//	        mac.Write([]byte("reflection-access"))
//	        expected := hex.EncodeToString(mac.Sum(nil))
//	        if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
//	            return errors.New("invalid signature")
//	        }
//	        return nil
//	    },
//	}
//
// # Client Usage
//
// Query the reflection service from a client:
//
//	packages, err := reflection.ListServices(ctx, conn)
//	for _, pkg := range packages {
//	    fmt.Printf("Package: %s\n", pkg.Name())
//	    for i := 0; i < pkg.ServicesLen(ctx); i++ {
//	        svc := pkg.ServicesGet(ctx, i)
//	        fmt.Printf("  Service: %s\n", svc.Name())
//	    }
//	}
package reflection

import (
	"fmt"
	"net"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
)

// TokenValidator validates an auth token from the request metadata.
// It receives the context and the raw token value from the AuthHeader.
// Return nil if the token is valid, or an error describing why it's invalid.
//
// The validator should use constant-time comparison (crypto/subtle.ConstantTimeCompare)
// when comparing tokens to prevent timing attacks.
//
// See the package documentation for examples of JWT validation, secret store
// integration, and HMAC verification.
type TokenValidator func(ctx context.Context, token string) error

// Config contains configuration for the reflection service access control.
// Both IP restrictions AND auth validation are checked (AND logic).
type Config struct {
	// AllowedCIDRs is a list of CIDR ranges that are allowed to access reflection.
	// Examples: "10.0.0.0/8", "192.168.0.0/16", "127.0.0.1/32"
	// If empty, all IPs are allowed (only auth restriction applies).
	AllowedCIDRs []string

	// AuthHeader is the metadata key name for the auth token.
	// Default is "authorization" if not specified.
	AuthHeader string

	// TokenValidator is called to validate tokens from the AuthHeader.
	// If nil, no auth validation is performed (only IP restriction applies).
	// This allows integration with JWT libraries, secret stores, etc.
	TokenValidator TokenValidator

	parsedCIDRs []*net.IPNet
	mu          sync.RWMutex
}

// Validate validates the configuration and parses CIDRs.
// Must be called before using the config.
func (c *Config) Validate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Parse CIDRs.
	c.parsedCIDRs = make([]*net.IPNet, 0, len(c.AllowedCIDRs))
	for _, cidr := range c.AllowedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}
		c.parsedCIDRs = append(c.parsedCIDRs, ipNet)
	}

	// Default auth header.
	if c.AuthHeader == "" {
		c.AuthHeader = "authorization"
	}

	return nil
}

// IsIPAllowed checks if the given IP is allowed by the CIDR restrictions.
// Returns true if no CIDR restrictions are configured (AllowedCIDRs is empty).
func (c *Config) IsIPAllowed(ip net.IP) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// No restrictions means all IPs are allowed.
	if len(c.parsedCIDRs) == 0 {
		return true
	}

	for _, ipNet := range c.parsedCIDRs {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// ValidateToken validates the token using the configured TokenValidator.
// Returns nil if no TokenValidator is configured (auth not required).
func (c *Config) ValidateToken(ctx context.Context, token string) error {
	c.mu.RLock()
	validator := c.TokenValidator
	c.mu.RUnlock()

	if validator == nil {
		return nil
	}

	return validator(ctx, token)
}
