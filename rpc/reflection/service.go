package reflection

import (
	"net"
	"sort"
	"strings"

	"github.com/gostdlib/base/context"

	rpcctx "github.com/bearlytools/claw/rpc/context"
	"github.com/bearlytools/claw/rpc/errors"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/server"
)

// Common errors.
var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrIPNotAllowed     = errors.New("IP address not allowed")
	ErrInvalidToken     = errors.New("invalid or missing auth token")
)

// Server implements the reflection service.
type Server struct {
	registry *server.Registry
	config   *Config
}

// NewServer creates a new reflection server.
// The config is validated during creation; returns an error if invalid.
func NewServer(registry *server.Registry, config Config) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Server{
		registry: registry,
		config:   &config,
	}, nil
}

// checkAccess verifies that the request is allowed based on IP and auth token.
// Both checks must pass (AND logic).
func (s *Server) checkAccess(ctx context.Context, md []msgs.Metadata) error {
	// Check IP restriction.
	remoteAddr := rpcctx.RemoteAddr(ctx)
	if remoteAddr != nil {
		ip := extractIP(remoteAddr)
		if ip != nil && !s.config.IsIPAllowed(ip) {
			return errors.E(ctx, errors.PermissionDenied, ErrIPNotAllowed)
		}
	}

	// Check auth token.
	if s.config.TokenValidator != nil {
		token := findMetadataValue(md, s.config.AuthHeader)
		if err := s.config.ValidateToken(ctx, token); err != nil {
			return errors.E(ctx, errors.Unauthenticated, ErrInvalidToken)
		}
	}

	return nil
}

// extractIP extracts the IP from a net.Addr.
func extractIP(addr net.Addr) net.IP {
	switch a := addr.(type) {
	case *net.TCPAddr:
		return a.IP
	case *net.UDPAddr:
		return a.IP
	case *net.IPAddr:
		return a.IP
	default:
		// Try to parse from string representation.
		host, _, err := net.SplitHostPort(addr.String())
		if err != nil {
			return net.ParseIP(addr.String())
		}
		return net.ParseIP(host)
	}
}

// findMetadataValue finds a value in metadata by key (case-insensitive).
func findMetadataValue(md []msgs.Metadata, key string) string {
	keyLower := strings.ToLower(key)
	for _, m := range md {
		if strings.ToLower(m.Key()) == keyLower {
			return string(m.Value())
		}
	}
	return ""
}

// ListServices handles the ListServices RPC.
func (s *Server) ListServices(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
	if err := s.checkAccess(ctx, md); err != nil {
		return nil, err
	}

	request := NewListServicesRequest(ctx)
	if err := request.Unmarshal(req); err != nil {
		return nil, err
	}

	// Collect handlers into package/service/method structure.
	type methodEntry struct {
		name    string
		rpcType msgs.RPCType
	}
	type serviceEntry struct {
		methods []methodEntry
	}
	type packageEntry struct {
		services map[string]*serviceEntry
	}

	packages := make(map[string]*packageEntry)

	for info := range s.registry.Handlers() {
		pkg, ok := packages[info.Package]
		if !ok {
			pkg = &packageEntry{services: make(map[string]*serviceEntry)}
			packages[info.Package] = pkg
		}

		svc, ok := pkg.services[info.Service]
		if !ok {
			svc = &serviceEntry{}
			pkg.services[info.Service] = svc
		}

		svc.methods = append(svc.methods, methodEntry{
			name:    info.Call,
			rpcType: info.Type,
		})
	}

	// Build response with sorted output for deterministic results.
	resp := NewListServicesResponse(ctx)

	pkgNames := make([]string, 0, len(packages))
	for name := range packages {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	for _, pkgName := range pkgNames {
		pkg := packages[pkgName]
		pkgInfo := NewPackageInfo(ctx)
		pkgInfo.SetName(pkgName)

		svcNames := make([]string, 0, len(pkg.services))
		for name := range pkg.services {
			svcNames = append(svcNames, name)
		}
		sort.Strings(svcNames)

		for _, svcName := range svcNames {
			svc := pkg.services[svcName]
			svcInfo := NewServiceInfo(ctx)
			svcInfo.SetName(svcName)

			// Sort methods by name.
			sort.Slice(svc.methods, func(i, j int) bool {
				return svc.methods[i].name < svc.methods[j].name
			})

			for _, method := range svc.methods {
				methodInfo := NewMethodInfo(ctx)
				methodInfo.SetName(method.name)
				methodInfo.SetType(method.rpcType)
				svcInfo.MethodsAppend(ctx, methodInfo)
			}

			pkgInfo.ServicesAppend(ctx, svcInfo)
		}

		resp.PackagesAppend(ctx, pkgInfo)
	}

	return resp.Marshal()
}

// GetServiceInfo handles the GetServiceInfo RPC.
func (s *Server) GetServiceInfo(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
	if err := s.checkAccess(ctx, md); err != nil {
		return nil, err
	}

	request := NewGetServiceInfoRequest(ctx)
	if err := request.Unmarshal(req); err != nil {
		return nil, err
	}

	pkgName := request.Package()
	svcName := request.Service()

	// Collect methods for this service.
	type methodEntry struct {
		name    string
		rpcType msgs.RPCType
	}
	var methods []methodEntry

	for info := range s.registry.Handlers() {
		if info.Package == pkgName && info.Service == svcName {
			methods = append(methods, methodEntry{
				name:    info.Call,
				rpcType: info.Type,
			})
		}
	}

	resp := NewGetServiceInfoResponse(ctx)

	if len(methods) == 0 {
		resp.SetFound(false)
		return resp.Marshal()
	}

	resp.SetFound(true)

	svcInfo := NewServiceInfo(ctx)
	svcInfo.SetName(svcName)

	// Sort methods for deterministic output.
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].name < methods[j].name
	})

	for _, method := range methods {
		methodInfo := NewMethodInfo(ctx)
		methodInfo.SetName(method.name)
		methodInfo.SetType(method.rpcType)
		svcInfo.MethodsAppend(ctx, methodInfo)
	}

	resp.SetService(svcInfo)

	return resp.Marshal()
}

// GetMethodInfo handles the GetMethodInfo RPC.
func (s *Server) GetMethodInfo(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
	if err := s.checkAccess(ctx, md); err != nil {
		return nil, err
	}

	request := NewGetMethodInfoRequest(ctx)
	if err := request.Unmarshal(req); err != nil {
		return nil, err
	}

	pkgName := request.Package()
	svcName := request.Service()
	methodName := request.Method()

	resp := NewGetMethodInfoResponse(ctx)

	for info := range s.registry.Handlers() {
		if info.Package == pkgName && info.Service == svcName && info.Call == methodName {
			resp.SetFound(true)
			methodInfo := NewMethodInfo(ctx)
			methodInfo.SetName(info.Call)
			methodInfo.SetType(info.Type)
			resp.SetMethod(methodInfo)
			return resp.Marshal()
		}
	}

	resp.SetFound(false)
	return resp.Marshal()
}

// ListServicesHandler returns a SyncHandler for the ListServices RPC.
func (s *Server) ListServicesHandler() server.SyncHandler {
	return server.SyncHandler{HandleFunc: s.ListServices}
}

// GetServiceInfoHandler returns a SyncHandler for the GetServiceInfo RPC.
func (s *Server) GetServiceInfoHandler() server.SyncHandler {
	return server.SyncHandler{HandleFunc: s.GetServiceInfo}
}

// GetMethodInfoHandler returns a SyncHandler for the GetMethodInfo RPC.
func (s *Server) GetMethodInfoHandler() server.SyncHandler {
	return server.SyncHandler{HandleFunc: s.GetMethodInfo}
}

// Register registers all reflection service RPCs with the server.
// The service is registered as "reflection/Reflection/{ListServices,GetServiceInfo,GetMethodInfo}".
func Register(ctx context.Context, srv *server.Server, reflectionSrv *Server) error {
	if err := srv.Register(ctx, "reflection", "Reflection", "ListServices", reflectionSrv.ListServicesHandler()); err != nil {
		return err
	}
	if err := srv.Register(ctx, "reflection", "Reflection", "GetServiceInfo", reflectionSrv.GetServiceInfoHandler()); err != nil {
		return err
	}
	if err := srv.Register(ctx, "reflection", "Reflection", "GetMethodInfo", reflectionSrv.GetMethodInfoHandler()); err != nil {
		return err
	}
	return nil
}

// Enable is a convenience function that creates a reflection server and registers it.
// Returns the reflection.Server for further use.
func Enable(ctx context.Context, srv *server.Server, config Config) (*Server, error) {
	reflectionSrv, err := NewServer(srv.Registry(), config)
	if err != nil {
		return nil, err
	}
	if err := Register(ctx, srv, reflectionSrv); err != nil {
		return nil, err
	}
	return reflectionSrv, nil
}
