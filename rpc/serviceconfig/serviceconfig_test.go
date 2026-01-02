package serviceconfig

import (
	"testing"
	"time"
)

func TestGetMethodConfigExactMatch(t *testing.T) {
	cfg := New().
		SetTimeout("myapp/UserService/GetUser", 5*time.Second)

	mc, ok := cfg.GetMethodConfig("myapp", "UserService", "GetUser")
	if !ok {
		t.Errorf("TestGetMethodConfigExactMatch: expected match, got none")
	}
	if mc.Timeout != 5*time.Second {
		t.Errorf("TestGetMethodConfigExactMatch: got timeout %v, want %v", mc.Timeout, 5*time.Second)
	}

	// Non-matching method should not match.
	_, ok = cfg.GetMethodConfig("myapp", "UserService", "DeleteUser")
	if ok {
		t.Errorf("TestGetMethodConfigExactMatch: expected no match for DeleteUser")
	}
}

func TestGetMethodConfigServiceWildcard(t *testing.T) {
	cfg := New().
		SetTimeout("myapp/UserService/*", 10*time.Second)

	tests := []struct {
		name    string
		pkg     string
		service string
		method  string
		want    bool
	}{
		{
			name:    "Success: matches GetUser",
			pkg:     "myapp",
			service: "UserService",
			method:  "GetUser",
			want:    true,
		},
		{
			name:    "Success: matches DeleteUser",
			pkg:     "myapp",
			service: "UserService",
			method:  "DeleteUser",
			want:    true,
		},
		{
			name:    "Success: no match for different service",
			pkg:     "myapp",
			service: "OrderService",
			method:  "GetOrder",
			want:    false,
		},
	}

	for _, test := range tests {
		mc, ok := cfg.GetMethodConfig(test.pkg, test.service, test.method)
		if ok != test.want {
			t.Errorf("TestGetMethodConfigServiceWildcard(%s): got ok=%v, want %v", test.name, ok, test.want)
		}
		if ok && mc.Timeout != 10*time.Second {
			t.Errorf("TestGetMethodConfigServiceWildcard(%s): got timeout %v, want %v", test.name, mc.Timeout, 10*time.Second)
		}
	}
}

func TestGetMethodConfigPackageWildcard(t *testing.T) {
	cfg := New().
		SetTimeout("myapp/*/*", 15*time.Second)

	tests := []struct {
		name    string
		pkg     string
		service string
		method  string
		want    bool
	}{
		{
			name:    "Success: matches UserService/GetUser",
			pkg:     "myapp",
			service: "UserService",
			method:  "GetUser",
			want:    true,
		},
		{
			name:    "Success: matches OrderService/GetOrder",
			pkg:     "myapp",
			service: "OrderService",
			method:  "GetOrder",
			want:    true,
		},
		{
			name:    "Success: no match for different package",
			pkg:     "otherapp",
			service: "UserService",
			method:  "GetUser",
			want:    false,
		},
	}

	for _, test := range tests {
		mc, ok := cfg.GetMethodConfig(test.pkg, test.service, test.method)
		if ok != test.want {
			t.Errorf("TestGetMethodConfigPackageWildcard(%s): got ok=%v, want %v", test.name, ok, test.want)
		}
		if ok && mc.Timeout != 15*time.Second {
			t.Errorf("TestGetMethodConfigPackageWildcard(%s): got timeout %v, want %v", test.name, mc.Timeout, 15*time.Second)
		}
	}
}

func TestGetMethodConfigGlobalWildcard(t *testing.T) {
	cfg := New().
		SetTimeout("*/*/*", 30*time.Second)

	mc, ok := cfg.GetMethodConfig("anyapp", "AnyService", "AnyMethod")
	if !ok {
		t.Errorf("TestGetMethodConfigGlobalWildcard: expected match, got none")
	}
	if mc.Timeout != 30*time.Second {
		t.Errorf("TestGetMethodConfigGlobalWildcard: got timeout %v, want %v", mc.Timeout, 30*time.Second)
	}
}

func TestGetMethodConfigPrecedence(t *testing.T) {
	cfg := New().
		SetTimeout("*/*/*", 30*time.Second).
		SetTimeout("myapp/*/*", 20*time.Second).
		SetTimeout("myapp/UserService/*", 10*time.Second).
		SetTimeout("myapp/UserService/GetUser", 5*time.Second)

	tests := []struct {
		name        string
		pkg         string
		service     string
		method      string
		wantTimeout time.Duration
	}{
		{
			name:        "Success: exact match takes precedence",
			pkg:         "myapp",
			service:     "UserService",
			method:      "GetUser",
			wantTimeout: 5 * time.Second,
		},
		{
			name:        "Success: service wildcard for other method",
			pkg:         "myapp",
			service:     "UserService",
			method:      "DeleteUser",
			wantTimeout: 10 * time.Second,
		},
		{
			name:        "Success: package wildcard for other service",
			pkg:         "myapp",
			service:     "OrderService",
			method:      "GetOrder",
			wantTimeout: 20 * time.Second,
		},
		{
			name:        "Success: global wildcard for other package",
			pkg:         "otherapp",
			service:     "SomeService",
			method:      "SomeMethod",
			wantTimeout: 30 * time.Second,
		},
	}

	for _, test := range tests {
		mc, ok := cfg.GetMethodConfig(test.pkg, test.service, test.method)
		if !ok {
			t.Errorf("TestGetMethodConfigPrecedence(%s): expected match, got none", test.name)
			continue
		}
		if mc.Timeout != test.wantTimeout {
			t.Errorf("TestGetMethodConfigPrecedence(%s): got timeout %v, want %v", test.name, mc.Timeout, test.wantTimeout)
		}
	}
}

func TestGetMethodConfigNilConfig(t *testing.T) {
	var cfg *Config
	_, ok := cfg.GetMethodConfig("pkg", "service", "method")
	if ok {
		t.Errorf("TestGetMethodConfigNilConfig: expected no match for nil config")
	}
}

func TestGetMethodConfigEmptyConfig(t *testing.T) {
	cfg := New()
	_, ok := cfg.GetMethodConfig("pkg", "service", "method")
	if ok {
		t.Errorf("TestGetMethodConfigEmptyConfig: expected no match for empty config")
	}
}

func TestWaitForReady(t *testing.T) {
	cfg := New().
		SetWaitForReady("myapp/UserService/*", true)

	mc, ok := cfg.GetMethodConfig("myapp", "UserService", "GetUser")
	if !ok {
		t.Errorf("TestWaitForReady: expected match, got none")
	}
	if !mc.WaitForReady {
		t.Errorf("TestWaitForReady: expected WaitForReady=true")
	}
}

func TestBuilder(t *testing.T) {
	cfg := NewBuilder().
		WithDefaultTimeout(30 * time.Second).
		WithTimeout("myapp/UserService/*", 10*time.Second).
		WithMethodConfig("myapp/UserService/SlowMethod", MethodConfig{
			Timeout:      60 * time.Second,
			WaitForReady: true,
		}).
		Build()

	// Check default timeout.
	timeout := cfg.GetTimeout("other", "Service", "Method")
	if timeout != 30*time.Second {
		t.Errorf("TestBuilder: default timeout got %v, want %v", timeout, 30*time.Second)
	}

	// Check service timeout.
	timeout = cfg.GetTimeout("myapp", "UserService", "GetUser")
	if timeout != 10*time.Second {
		t.Errorf("TestBuilder: service timeout got %v, want %v", timeout, 10*time.Second)
	}

	// Check method config.
	mc, ok := cfg.GetMethodConfig("myapp", "UserService", "SlowMethod")
	if !ok {
		t.Errorf("TestBuilder: expected match for SlowMethod")
	}
	if mc.Timeout != 60*time.Second {
		t.Errorf("TestBuilder: method timeout got %v, want %v", mc.Timeout, 60*time.Second)
	}
	if !mc.WaitForReady {
		t.Errorf("TestBuilder: expected WaitForReady=true for SlowMethod")
	}
}

func TestParsePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantPkg string
		wantSvc string
		wantMth string
		wantOK  bool
	}{
		{
			name:    "Success: valid pattern",
			pattern: "myapp/UserService/GetUser",
			wantPkg: "myapp",
			wantSvc: "UserService",
			wantMth: "GetUser",
			wantOK:  true,
		},
		{
			name:    "Success: wildcard pattern",
			pattern: "myapp/UserService/*",
			wantPkg: "myapp",
			wantSvc: "UserService",
			wantMth: "*",
			wantOK:  true,
		},
		{
			name:    "Success: global wildcard",
			pattern: "*/*/*",
			wantPkg: "*",
			wantSvc: "*",
			wantMth: "*",
			wantOK:  true,
		},
		{
			name:    "Error: too few parts",
			pattern: "myapp/UserService",
			wantOK:  false,
		},
		{
			name:    "Error: too many parts",
			pattern: "myapp/UserService/GetUser/extra",
			wantOK:  false,
		},
	}

	for _, test := range tests {
		pkg, svc, mth, ok := ParsePattern(test.pattern)
		if ok != test.wantOK {
			t.Errorf("TestParsePattern(%s): got ok=%v, want %v", test.name, ok, test.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if pkg != test.wantPkg || svc != test.wantSvc || mth != test.wantMth {
			t.Errorf("TestParsePattern(%s): got (%s, %s, %s), want (%s, %s, %s)",
				test.name, pkg, svc, mth, test.wantPkg, test.wantSvc, test.wantMth)
		}
	}
}
