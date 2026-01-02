package reflection

import (
	"net"
	"testing"

	"github.com/gostdlib/base/context"
	"github.com/kylelemons/godebug/pretty"

	rpcctx "github.com/bearlytools/claw/rpc/context"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/server"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "Success: empty config",
			config:  Config{},
			wantErr: false,
		},
		{
			name: "Success: valid CIDRs",
			config: Config{
				AllowedCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16", "127.0.0.1/32"},
			},
			wantErr: false,
		},
		{
			name: "Success: with token validator",
			config: Config{
				TokenValidator: func(ctx context.Context, token string) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "Success: custom auth header",
			config: Config{
				AuthHeader: "x-api-key",
				TokenValidator: func(ctx context.Context, token string) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "Error: invalid CIDR",
			config: Config{
				AllowedCIDRs: []string{"not-a-cidr"},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := test.config.Validate()
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestConfigValidate(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestConfigValidate(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestConfigIsIPAllowed(t *testing.T) {
	tests := []struct {
		name   string
		cidrs  []string
		ip     string
		want   bool
	}{
		{
			name:  "Success: no restrictions allows all",
			cidrs: nil,
			ip:    "1.2.3.4",
			want:  true,
		},
		{
			name:  "Success: IP in allowed range",
			cidrs: []string{"10.0.0.0/8"},
			ip:    "10.1.2.3",
			want:  true,
		},
		{
			name:  "Success: IP not in allowed range",
			cidrs: []string{"10.0.0.0/8"},
			ip:    "192.168.1.1",
			want:  false,
		},
		{
			name:  "Success: IP in one of multiple ranges",
			cidrs: []string{"10.0.0.0/8", "192.168.0.0/16"},
			ip:    "192.168.1.1",
			want:  true,
		},
		{
			name:  "Success: exact IP match",
			cidrs: []string{"127.0.0.1/32"},
			ip:    "127.0.0.1",
			want:  true,
		},
		{
			name:  "Success: exact IP not match",
			cidrs: []string{"127.0.0.1/32"},
			ip:    "127.0.0.2",
			want:  false,
		},
	}

	for _, test := range tests {
		config := Config{AllowedCIDRs: test.cidrs}
		if err := config.Validate(); err != nil {
			t.Fatalf("TestConfigIsIPAllowed(%s): Validate failed: %v", test.name, err)
		}

		ip := net.ParseIP(test.ip)
		got := config.IsIPAllowed(ip)
		if got != test.want {
			t.Errorf("TestConfigIsIPAllowed(%s): got %v, want %v", test.name, got, test.want)
		}
	}
}

func TestConfigValidateToken(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name      string
		validator TokenValidator
		token     string
		wantErr   bool
	}{
		{
			name:      "Success: no validator means all valid",
			validator: nil,
			token:     "anything",
			wantErr:   false,
		},
		{
			name: "Success: valid token",
			validator: func(ctx context.Context, token string) error {
				if token == "secret" {
					return nil
				}
				return ErrInvalidToken
			},
			token:   "secret",
			wantErr: false,
		},
		{
			name: "Error: invalid token",
			validator: func(ctx context.Context, token string) error {
				if token == "secret" {
					return nil
				}
				return ErrInvalidToken
			},
			token:   "wrong",
			wantErr: true,
		},
		{
			name: "Error: empty token when validator configured",
			validator: func(ctx context.Context, token string) error {
				if token == "" {
					return ErrInvalidToken
				}
				return nil
			},
			token:   "",
			wantErr: true,
		},
	}

	for _, test := range tests {
		config := Config{TokenValidator: test.validator}
		if err := config.Validate(); err != nil {
			t.Fatalf("TestConfigValidateToken(%s): Validate failed: %v", test.name, err)
		}

		err := config.ValidateToken(ctx, test.token)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestConfigValidateToken(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestConfigValidateToken(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestCheckAccessIPRestriction(t *testing.T) {
	ctx := t.Context()

	config := Config{
		AllowedCIDRs: []string{"10.0.0.0/8"},
	}
	registry := server.NewRegistry()
	srv, err := NewServer(registry, config)
	if err != nil {
		t.Fatalf("TestCheckAccessIPRestriction: NewServer failed: %v", err)
	}

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{
			name:    "Success: allowed IP",
			ip:      "10.1.2.3",
			wantErr: false,
		},
		{
			name:    "Error: disallowed IP",
			ip:      "192.168.1.1",
			wantErr: true,
		},
	}

	for _, test := range tests {
		testCtx := rpcctx.WithRemoteAddr(ctx, &net.TCPAddr{IP: net.ParseIP(test.ip)})
		err := srv.checkAccess(testCtx, nil)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestCheckAccessIPRestriction(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestCheckAccessIPRestriction(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestCheckAccessAuthRestriction(t *testing.T) {
	ctx := t.Context()

	config := Config{
		AuthHeader: "authorization",
		TokenValidator: func(ctx context.Context, token string) error {
			if token == "Bearer secret" {
				return nil
			}
			return ErrInvalidToken
		},
	}
	registry := server.NewRegistry()
	srv, err := NewServer(registry, config)
	if err != nil {
		t.Fatalf("TestCheckAccessAuthRestriction: NewServer failed: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "Success: valid token",
			token:   "Bearer secret",
			wantErr: false,
		},
		{
			name:    "Error: invalid token",
			token:   "Bearer wrong",
			wantErr: true,
		},
		{
			name:    "Error: missing token",
			token:   "",
			wantErr: true,
		},
	}

	for _, test := range tests {
		var md []msgs.Metadata
		if test.token != "" {
			m := msgs.NewMetadata(ctx)
			m.SetKey("authorization")
			m.SetValue([]byte(test.token))
			md = []msgs.Metadata{m}
		}

		err := srv.checkAccess(ctx, md)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestCheckAccessAuthRestriction(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestCheckAccessAuthRestriction(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestCheckAccessBothRestrictions(t *testing.T) {
	ctx := t.Context()

	config := Config{
		AllowedCIDRs: []string{"10.0.0.0/8"},
		AuthHeader:   "authorization",
		TokenValidator: func(ctx context.Context, token string) error {
			if token == "Bearer secret" {
				return nil
			}
			return ErrInvalidToken
		},
	}
	registry := server.NewRegistry()
	srv, err := NewServer(registry, config)
	if err != nil {
		t.Fatalf("TestCheckAccessBothRestrictions: NewServer failed: %v", err)
	}

	tests := []struct {
		name    string
		ip      string
		token   string
		wantErr bool
	}{
		{
			name:    "Success: both valid",
			ip:      "10.1.2.3",
			token:   "Bearer secret",
			wantErr: false,
		},
		{
			name:    "Error: valid IP, invalid token",
			ip:      "10.1.2.3",
			token:   "Bearer wrong",
			wantErr: true,
		},
		{
			name:    "Error: invalid IP, valid token",
			ip:      "192.168.1.1",
			token:   "Bearer secret",
			wantErr: true,
		},
		{
			name:    "Error: both invalid",
			ip:      "192.168.1.1",
			token:   "Bearer wrong",
			wantErr: true,
		},
	}

	for _, test := range tests {
		testCtx := rpcctx.WithRemoteAddr(ctx, &net.TCPAddr{IP: net.ParseIP(test.ip)})
		var md []msgs.Metadata
		if test.token != "" {
			m := msgs.NewMetadata(ctx)
			m.SetKey("authorization")
			m.SetValue([]byte(test.token))
			md = []msgs.Metadata{m}
		}

		err := srv.checkAccess(testCtx, md)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestCheckAccessBothRestrictions(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestCheckAccessBothRestrictions(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestListServices(t *testing.T) {
	ctx := t.Context()

	registry := server.NewRegistry()

	// Register some handlers.
	syncHandler := server.SyncHandler{HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
		return nil, nil
	}}
	biDirHandler := server.BiDirHandler{HandleFunc: func(ctx context.Context, stream *server.BiDirStream) error {
		return nil
	}}

	registry.Register(ctx, "myapp", "UserService", "GetUser", syncHandler)
	registry.Register(ctx, "myapp", "UserService", "CreateUser", syncHandler)
	registry.Register(ctx, "myapp", "OrderService", "ListOrders", biDirHandler)
	registry.Register(ctx, "other", "SomeService", "DoThing", syncHandler)

	config := Config{}
	srv, err := NewServer(registry, config)
	if err != nil {
		t.Fatalf("TestListServices: NewServer failed: %v", err)
	}

	req := NewListServicesRequest(ctx)
	reqBytes, err := req.Marshal()
	if err != nil {
		t.Fatalf("TestListServices: Marshal failed: %v", err)
	}

	respBytes, err := srv.ListServices(ctx, reqBytes, nil)
	if err != nil {
		t.Fatalf("TestListServices: ListServices failed: %v", err)
	}

	resp := NewListServicesResponse(ctx)
	if err := resp.Unmarshal(respBytes); err != nil {
		t.Fatalf("TestListServices: Unmarshal failed: %v", err)
	}

	// Verify packages.
	if resp.PackagesLen(ctx) != 2 {
		t.Fatalf("TestListServices: got %d packages, want 2", resp.PackagesLen(ctx))
	}

	// First package should be "myapp" (alphabetically).
	pkg0 := resp.PackagesGet(ctx, 0)
	if pkg0.Name() != "myapp" {
		t.Errorf("TestListServices: got package[0].Name = %q, want %q", pkg0.Name(), "myapp")
	}
	if pkg0.ServicesLen(ctx) != 2 {
		t.Errorf("TestListServices: got %d services in myapp, want 2", pkg0.ServicesLen(ctx))
	}

	// Check services in myapp.
	svc0 := pkg0.ServicesGet(ctx, 0) // OrderService
	if svc0.Name() != "OrderService" {
		t.Errorf("TestListServices: got service[0].Name = %q, want %q", svc0.Name(), "OrderService")
	}
	if svc0.MethodsLen(ctx) != 1 {
		t.Errorf("TestListServices: got %d methods in OrderService, want 1", svc0.MethodsLen(ctx))
	}

	svc1 := pkg0.ServicesGet(ctx, 1) // UserService
	if svc1.Name() != "UserService" {
		t.Errorf("TestListServices: got service[1].Name = %q, want %q", svc1.Name(), "UserService")
	}
	if svc1.MethodsLen(ctx) != 2 {
		t.Errorf("TestListServices: got %d methods in UserService, want 2", svc1.MethodsLen(ctx))
	}

	// Check method types.
	method0 := svc0.MethodsGet(ctx, 0)
	if method0.Name() != "ListOrders" {
		t.Errorf("TestListServices: got method name %q, want %q", method0.Name(), "ListOrders")
	}
	if method0.Type() != msgs.RTBiDirectional {
		t.Errorf("TestListServices: got method type %v, want %v", method0.Type(), msgs.RTBiDirectional)
	}

	// Check second package.
	pkg1 := resp.PackagesGet(ctx, 1)
	if pkg1.Name() != "other" {
		t.Errorf("TestListServices: got package[1].Name = %q, want %q", pkg1.Name(), "other")
	}
}

func TestGetServiceInfo(t *testing.T) {
	ctx := t.Context()

	registry := server.NewRegistry()

	syncHandler := server.SyncHandler{HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
		return nil, nil
	}}

	registry.Register(ctx, "myapp", "UserService", "GetUser", syncHandler)
	registry.Register(ctx, "myapp", "UserService", "CreateUser", syncHandler)

	config := Config{}
	srv, err := NewServer(registry, config)
	if err != nil {
		t.Fatalf("TestGetServiceInfo: NewServer failed: %v", err)
	}

	tests := []struct {
		name      string
		pkg       string
		service   string
		wantFound bool
		wantCount int
	}{
		{
			name:      "Success: existing service",
			pkg:       "myapp",
			service:   "UserService",
			wantFound: true,
			wantCount: 2,
		},
		{
			name:      "Success: non-existent service",
			pkg:       "myapp",
			service:   "NonExistent",
			wantFound: false,
			wantCount: 0,
		},
		{
			name:      "Success: non-existent package",
			pkg:       "nonexistent",
			service:   "UserService",
			wantFound: false,
			wantCount: 0,
		},
	}

	for _, test := range tests {
		req := NewGetServiceInfoRequest(ctx).SetPackage(test.pkg).SetService(test.service)
		reqBytes, err := req.Marshal()
		if err != nil {
			t.Fatalf("TestGetServiceInfo(%s): Marshal failed: %v", test.name, err)
		}

		respBytes, err := srv.GetServiceInfo(ctx, reqBytes, nil)
		if err != nil {
			t.Fatalf("TestGetServiceInfo(%s): GetServiceInfo failed: %v", test.name, err)
		}

		resp := NewGetServiceInfoResponse(ctx)
		if err := resp.Unmarshal(respBytes); err != nil {
			t.Fatalf("TestGetServiceInfo(%s): Unmarshal failed: %v", test.name, err)
		}

		if resp.Found() != test.wantFound {
			t.Errorf("TestGetServiceInfo(%s): got Found = %v, want %v", test.name, resp.Found(), test.wantFound)
		}

		if test.wantFound {
			svc := resp.Service()
			if svc.MethodsLen(ctx) != test.wantCount {
				t.Errorf("TestGetServiceInfo(%s): got %d methods, want %d", test.name, svc.MethodsLen(ctx), test.wantCount)
			}
		}
	}
}

func TestGetMethodInfo(t *testing.T) {
	ctx := t.Context()

	registry := server.NewRegistry()

	syncHandler := server.SyncHandler{HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
		return nil, nil
	}}
	biDirHandler := server.BiDirHandler{HandleFunc: func(ctx context.Context, stream *server.BiDirStream) error {
		return nil
	}}

	registry.Register(ctx, "myapp", "UserService", "GetUser", syncHandler)
	registry.Register(ctx, "myapp", "UserService", "Stream", biDirHandler)

	config := Config{}
	srv, err := NewServer(registry, config)
	if err != nil {
		t.Fatalf("TestGetMethodInfo: NewServer failed: %v", err)
	}

	tests := []struct {
		name      string
		pkg       string
		service   string
		method    string
		wantFound bool
		wantType  msgs.RPCType
	}{
		{
			name:      "Success: existing sync method",
			pkg:       "myapp",
			service:   "UserService",
			method:    "GetUser",
			wantFound: true,
			wantType:  msgs.RTSynchronous,
		},
		{
			name:      "Success: existing bidir method",
			pkg:       "myapp",
			service:   "UserService",
			method:    "Stream",
			wantFound: true,
			wantType:  msgs.RTBiDirectional,
		},
		{
			name:      "Success: non-existent method",
			pkg:       "myapp",
			service:   "UserService",
			method:    "NonExistent",
			wantFound: false,
		},
	}

	for _, test := range tests {
		req := NewGetMethodInfoRequest(ctx).SetPackage(test.pkg).SetService(test.service).SetMethod(test.method)
		reqBytes, err := req.Marshal()
		if err != nil {
			t.Fatalf("TestGetMethodInfo(%s): Marshal failed: %v", test.name, err)
		}

		respBytes, err := srv.GetMethodInfo(ctx, reqBytes, nil)
		if err != nil {
			t.Fatalf("TestGetMethodInfo(%s): GetMethodInfo failed: %v", test.name, err)
		}

		resp := NewGetMethodInfoResponse(ctx)
		if err := resp.Unmarshal(respBytes); err != nil {
			t.Fatalf("TestGetMethodInfo(%s): Unmarshal failed: %v", test.name, err)
		}

		if resp.Found() != test.wantFound {
			t.Errorf("TestGetMethodInfo(%s): got Found = %v, want %v", test.name, resp.Found(), test.wantFound)
		}

		if test.wantFound {
			method := resp.Method()
			if method.Type() != test.wantType {
				t.Errorf("TestGetMethodInfo(%s): got Type = %v, want %v", test.name, method.Type(), test.wantType)
			}
		}
	}
}

func TestRegister(t *testing.T) {
	ctx := t.Context()
	srv := server.New()
	registry := srv.Registry()

	config := Config{}
	reflectionSrv, err := NewServer(registry, config)
	if err != nil {
		t.Fatalf("TestRegister: NewServer failed: %v", err)
	}

	if err := Register(ctx, srv, reflectionSrv); err != nil {
		t.Fatalf("TestRegister: Register failed: %v", err)
	}

	// Verify handlers are registered.
	handlers := []struct {
		pkg     string
		service string
		call    string
	}{
		{"reflection", "Reflection", "ListServices"},
		{"reflection", "Reflection", "GetServiceInfo"},
		{"reflection", "Reflection", "GetMethodInfo"},
	}

	for _, h := range handlers {
		_, found := registry.Lookup(h.pkg, h.service, h.call)
		if !found {
			t.Errorf("TestRegister: handler not found: %s/%s/%s", h.pkg, h.service, h.call)
		}
	}
}

func TestEnable(t *testing.T) {
	ctx := t.Context()
	srv := server.New()

	config := Config{
		AllowedCIDRs: []string{"10.0.0.0/8"},
		TokenValidator: func(ctx context.Context, token string) error {
			if token == "secret" {
				return nil
			}
			return ErrInvalidToken
		},
	}

	reflectionSrv, err := Enable(ctx, srv, config)
	if err != nil {
		t.Fatalf("TestEnable: Enable failed: %v", err)
	}

	if reflectionSrv == nil {
		t.Fatal("TestEnable: got nil server, want non-nil")
	}

	// Verify it's registered.
	_, found := srv.Registry().Lookup("reflection", "Reflection", "ListServices")
	if !found {
		t.Error("TestEnable: ListServices not registered")
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name string
		addr net.Addr
		want string
	}{
		{
			name: "Success: TCPAddr",
			addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 8080},
			want: "192.168.1.1",
		},
		{
			name: "Success: UDPAddr",
			addr: &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 1234},
			want: "10.0.0.1",
		},
		{
			name: "Success: IPAddr",
			addr: &net.IPAddr{IP: net.ParseIP("172.16.0.1")},
			want: "172.16.0.1",
		},
	}

	for _, test := range tests {
		got := extractIP(test.addr)
		if diff := pretty.Compare(got.String(), test.want); diff != "" {
			t.Errorf("TestExtractIP(%s): diff (-got +want):\n%s", test.name, diff)
		}
	}
}
