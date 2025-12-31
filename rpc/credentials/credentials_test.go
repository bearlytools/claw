package credentials

import (
	"errors"
	"testing"

	"github.com/gostdlib/base/context"
	"github.com/kylelemons/godebug/pretty"
)

func TestTokenCredentials(t *testing.T) {
	tests := []struct {
		name        string
		tokenType   string
		token       string
		requireSec  bool
		wantAuth    string
		wantRequire bool
	}{
		{
			name:        "Success: bearer token",
			tokenType:   "Bearer",
			token:       "secret123",
			requireSec:  true,
			wantAuth:    "Bearer secret123",
			wantRequire: true,
		},
		{
			name:        "Success: no token type",
			tokenType:   "",
			token:       "api-key-value",
			requireSec:  false,
			wantAuth:    "api-key-value",
			wantRequire: false,
		},
		{
			name:        "Success: basic auth",
			tokenType:   "Basic",
			token:       "dXNlcjpwYXNz",
			requireSec:  true,
			wantAuth:    "Basic dXNlcjpwYXNz",
			wantRequire: true,
		},
	}

	for _, test := range tests {
		ctx := t.Context()
		creds := NewTokenCredentials(test.tokenType, test.token, test.requireSec)

		md, err := creds.GetRequestMetadata(ctx, "pkg/svc/method")
		if err != nil {
			t.Errorf("[TestTokenCredentials](%s): got err = %v, want nil", test.name, err)
			continue
		}

		if md["authorization"] != test.wantAuth {
			t.Errorf("[TestTokenCredentials](%s): auth = %q, want %q", test.name, md["authorization"], test.wantAuth)
		}

		if creds.RequireTransportSecurity() != test.wantRequire {
			t.Errorf("[TestTokenCredentials](%s): requireSecurity = %v, want %v", test.name, creds.RequireTransportSecurity(), test.wantRequire)
		}
	}
}

type fakeTokenSource struct {
	token string
	err   error
}

func (f *fakeTokenSource) Token(ctx context.Context) (string, error) {
	return f.token, f.err
}

func TestTokenSourceCredentials(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name      string
		source    *fakeTokenSource
		tokenType string
		wantAuth  string
		wantErr   bool
	}{
		{
			name:      "Success: bearer token",
			source:    &fakeTokenSource{token: "dynamic-token"},
			tokenType: "Bearer",
			wantAuth:  "Bearer dynamic-token",
		},
		{
			name:      "Success: no token type",
			source:    &fakeTokenSource{token: "raw-token"},
			tokenType: "",
			wantAuth:  "raw-token",
		},
		{
			name:      "Error: token source fails",
			source:    &fakeTokenSource{err: errors.New("token expired")},
			tokenType: "Bearer",
			wantErr:   true,
		},
	}

	for _, test := range tests {
		source := &tokenSourceAdapter{test.source}
		creds := NewTokenSourceCredentials(test.tokenType, source, true)

		md, err := creds.GetRequestMetadata(ctx, "pkg/svc/method")
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestTokenSourceCredentials](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestTokenSourceCredentials](%s): got err = %v, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if md["authorization"] != test.wantAuth {
			t.Errorf("[TestTokenSourceCredentials](%s): auth = %q, want %q", test.name, md["authorization"], test.wantAuth)
		}
	}
}

// tokenSourceAdapter adapts fakeTokenSource to TokenSource
type tokenSourceAdapter struct {
	*fakeTokenSource
}

func (a *tokenSourceAdapter) Token(ctx context.Context) (string, error) {
	return a.fakeTokenSource.Token(ctx)
}

func TestAPIKeyCredentials(t *testing.T) {
	ctx := t.Context()

	creds := NewAPIKeyCredentials("x-api-key", "my-api-key-123", false)

	md, err := creds.GetRequestMetadata(ctx, "pkg/svc/method")
	if err != nil {
		t.Errorf("[TestAPIKeyCredentials]: got err = %v, want nil", err)
		return
	}

	if md["x-api-key"] != "my-api-key-123" {
		t.Errorf("[TestAPIKeyCredentials]: x-api-key = %q, want %q", md["x-api-key"], "my-api-key-123")
	}

	if creds.RequireTransportSecurity() {
		t.Error("[TestAPIKeyCredentials]: requireSecurity = true, want false")
	}
}

func TestCompositeCredentials(t *testing.T) {
	ctx := t.Context()

	tokenCreds := NewTokenCredentials("Bearer", "token123", true)
	apiKeyCreds := NewAPIKeyCredentials("x-api-key", "apikey456", false)

	composite := NewCompositeCredentials(tokenCreds, apiKeyCreds)

	md, err := composite.GetRequestMetadata(ctx, "pkg/svc/method")
	if err != nil {
		t.Errorf("[TestCompositeCredentials]: got err = %v, want nil", err)
		return
	}

	want := map[string]string{
		"authorization": "Bearer token123",
		"x-api-key":     "apikey456",
	}

	if diff := pretty.Compare(want, md); diff != "" {
		t.Errorf("[TestCompositeCredentials]: mismatch (-want +got):\n%s", diff)
	}

	// Should require security because tokenCreds requires it.
	if !composite.RequireTransportSecurity() {
		t.Error("[TestCompositeCredentials]: requireSecurity = false, want true")
	}
}

func TestCompositeCredentialsNoSecurity(t *testing.T) {
	apiKeyCreds1 := NewAPIKeyCredentials("x-api-key", "key1", false)
	apiKeyCreds2 := NewAPIKeyCredentials("x-other-key", "key2", false)

	composite := NewCompositeCredentials(apiKeyCreds1, apiKeyCreds2)

	if composite.RequireTransportSecurity() {
		t.Error("[TestCompositeCredentialsNoSecurity]: requireSecurity = true, want false")
	}
}
