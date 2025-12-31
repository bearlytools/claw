package health

import (
	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/server"
)

// Server implements the health check service.
// Use NewServer() to create an instance, then register it with an RPC server
// using Register() or server.EnableHealthCheck().
type Server struct {
	mu       sync.RWMutex
	services map[string]ServingStatus
}

// NewServer creates a new health check server.
// By default, the overall server health (empty service name) is set to Serving.
func NewServer() *Server {
	return &Server{
		services: map[string]ServingStatus{
			"": Serving, // Empty string = overall server health
		},
	}
}

// SetServingStatus sets the health status for a service.
// Use an empty string to set the overall server health status.
func (s *Server) SetServingStatus(service string, status ServingStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[service] = status
}

// ServingStatus returns the health status for a service.
// Returns ServiceUnknown if the service is not registered.
func (s *Server) ServingStatus(service string) ServingStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.services[service]
	if !ok {
		return ServiceUnknown
	}
	return status
}

// Check handles a health check request.
func (s *Server) Check(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
	request := NewHealthCheckRequest(ctx)
	if err := request.Unmarshal(req); err != nil {
		return nil, err
	}

	s.mu.RLock()
	status, ok := s.services[request.Service()]
	s.mu.RUnlock()

	resp := NewHealthCheckResponse(ctx)
	if !ok {
		resp.SetStatus(ServiceUnknown)
	} else {
		resp.SetStatus(status)
	}

	return resp.Marshal()
}

// Handler returns a SyncHandler for the health check service.
func (s *Server) Handler() server.SyncHandler {
	return server.SyncHandler{HandleFunc: s.Check}
}

// Register registers the health check service with an RPC server.
// The service is registered as "health/Health/Check".
// This is the primary way to enable health checking on a server.
//
// Example usage:
//
//	srv := server.New()
//	healthSvc := health.NewServer()
//	health.Register(srv, healthSvc)
//	// Now srv has a health endpoint at health/Health/Check
//	// Update service status as needed:
//	healthSvc.SetServingStatus("myservice", health.Serving)
func Register(srv *server.Server, health *Server) error {
	return srv.Register("health", "Health", "Check", health.Handler())
}

// Enable is a convenience function that creates a health server and registers it.
// Returns the health.Server so you can update service status.
//
// Example:
//
//	srv := server.New()
//	healthSvc := health.Enable(srv)
//	healthSvc.SetServingStatus("myservice", health.Serving)
func Enable(srv *server.Server) (*Server, error) {
	h := NewServer()
	if err := Register(srv, h); err != nil {
		return nil, err
	}
	return h, nil
}
