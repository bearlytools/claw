// Package credentials provides common PerRPCCredentials implementations.
package credentials

import (
	"github.com/gostdlib/base/context"
)

// TokenCredentials provides static token credentials for RPC calls.
// The token is sent as an "authorization" metadata header.
type TokenCredentials struct {
	token                    string
	requireTransportSecurity bool
}

// NewTokenCredentials creates credentials that attach a static token to each call.
// The tokenType is typically "Bearer" for bearer tokens.
// Example: NewTokenCredentials("Bearer", "my-secret-token", true)
func NewTokenCredentials(tokenType, token string, requireTransportSecurity bool) *TokenCredentials {
	t := token
	if tokenType != "" {
		t = tokenType + " " + token
	}
	return &TokenCredentials{
		token:                    t,
		requireTransportSecurity: requireTransportSecurity,
	}
}

// GetRequestMetadata returns the authorization header metadata.
func (t *TokenCredentials) GetRequestMetadata(ctx context.Context, uri string) (map[string]string, error) {
	return map[string]string{
		"authorization": t.token,
	}, nil
}

// RequireTransportSecurity returns whether TLS is required.
func (t *TokenCredentials) RequireTransportSecurity() bool {
	return t.requireTransportSecurity
}

// TokenSource provides tokens dynamically.
// Implementations can refresh tokens, fetch from a secrets manager, etc.
type TokenSource interface {
	// Token returns the current token and any error.
	// This may be called for each RPC, so implementations should cache
	// tokens appropriately.
	Token(ctx context.Context) (string, error)
}

// TokenSourceCredentials provides credentials backed by a TokenSource.
// This allows for dynamic token refresh (e.g., OAuth2, rotating secrets).
type TokenSourceCredentials struct {
	source                   TokenSource
	tokenType                string
	requireTransportSecurity bool
}

// NewTokenSourceCredentials creates credentials that fetch tokens dynamically.
// The tokenType is prepended to the token (e.g., "Bearer").
func NewTokenSourceCredentials(tokenType string, source TokenSource, requireTransportSecurity bool) *TokenSourceCredentials {
	return &TokenSourceCredentials{
		source:                   source,
		tokenType:                tokenType,
		requireTransportSecurity: requireTransportSecurity,
	}
}

// GetRequestMetadata fetches a token and returns authorization metadata.
func (t *TokenSourceCredentials) GetRequestMetadata(ctx context.Context, uri string) (map[string]string, error) {
	token, err := t.source.Token(ctx)
	if err != nil {
		return nil, err
	}

	authValue := token
	if t.tokenType != "" {
		authValue = t.tokenType + " " + token
	}

	return map[string]string{
		"authorization": authValue,
	}, nil
}

// RequireTransportSecurity returns whether TLS is required.
func (t *TokenSourceCredentials) RequireTransportSecurity() bool {
	return t.requireTransportSecurity
}

// APIKeyCredentials provides API key credentials.
// The key is sent as a custom metadata header.
type APIKeyCredentials struct {
	headerName               string
	apiKey                   string
	requireTransportSecurity bool
}

// NewAPIKeyCredentials creates credentials that attach an API key.
// headerName is the metadata key (e.g., "x-api-key", "api-key").
func NewAPIKeyCredentials(headerName, apiKey string, requireTransportSecurity bool) *APIKeyCredentials {
	return &APIKeyCredentials{
		headerName:               headerName,
		apiKey:                   apiKey,
		requireTransportSecurity: requireTransportSecurity,
	}
}

// GetRequestMetadata returns the API key header metadata.
func (a *APIKeyCredentials) GetRequestMetadata(ctx context.Context, uri string) (map[string]string, error) {
	return map[string]string{
		a.headerName: a.apiKey,
	}, nil
}

// RequireTransportSecurity returns whether TLS is required.
func (a *APIKeyCredentials) RequireTransportSecurity() bool {
	return a.requireTransportSecurity
}

// CompositeCredentials combines multiple credentials.
// All credential metadata is merged (later credentials override earlier on conflicts).
type CompositeCredentials struct {
	creds                    []interface{ GetRequestMetadata(context.Context, string) (map[string]string, error); RequireTransportSecurity() bool }
	requireTransportSecurity bool
}

// NewCompositeCredentials creates credentials that combine multiple sources.
// Security requirement is true if any component requires it.
func NewCompositeCredentials(creds ...interface{ GetRequestMetadata(context.Context, string) (map[string]string, error); RequireTransportSecurity() bool }) *CompositeCredentials {
	requireSecurity := false
	for _, c := range creds {
		if c.RequireTransportSecurity() {
			requireSecurity = true
			break
		}
	}
	return &CompositeCredentials{
		creds:                    creds,
		requireTransportSecurity: requireSecurity,
	}
}

// GetRequestMetadata merges metadata from all credentials.
func (c *CompositeCredentials) GetRequestMetadata(ctx context.Context, uri string) (map[string]string, error) {
	result := make(map[string]string)
	for _, cred := range c.creds {
		md, err := cred.GetRequestMetadata(ctx, uri)
		if err != nil {
			return nil, err
		}
		for k, v := range md {
			result[k] = v
		}
	}
	return result, nil
}

// RequireTransportSecurity returns true if any component requires security.
func (c *CompositeCredentials) RequireTransportSecurity() bool {
	return c.requireTransportSecurity
}
