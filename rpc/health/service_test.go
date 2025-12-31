package health

import (
	"testing"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

func TestNewServer(t *testing.T) {
	srv := NewServer()
	if srv == nil {
		t.Error("[TestNewServer]: got nil, want non-nil server")
		return
	}

	// Check default status
	status := srv.ServingStatus("")
	if status != Serving {
		t.Errorf("[TestNewServer]: got default status = %v, want %v", status, Serving)
	}
}

func TestServerSetServingStatus(t *testing.T) {
	tests := []struct {
		name    string
		service string
		status  ServingStatus
	}{
		{
			name:    "Success: set overall health",
			service: "",
			status:  NotServing,
		},
		{
			name:    "Success: set specific service",
			service: "myservice",
			status:  Serving,
		},
		{
			name:    "Success: set unknown status",
			service: "another",
			status:  Unknown,
		},
	}

	for _, test := range tests {
		srv := NewServer()
		srv.SetServingStatus(test.service, test.status)

		got := srv.ServingStatus(test.service)
		if got != test.status {
			t.Errorf("[TestServerSetServingStatus](%s): got status = %v, want %v", test.name, got, test.status)
		}
	}
}

func TestServerServingStatusUnknownService(t *testing.T) {
	srv := NewServer()

	// Query a service that was never registered
	status := srv.ServingStatus("nonexistent")
	if status != ServiceUnknown {
		t.Errorf("[TestServerServingStatusUnknownService]: got status = %v, want %v", status, ServiceUnknown)
	}
}

func TestServerCheck(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name       string
		setup      func(*Server)
		service    string
		wantStatus ServingStatus
		wantErr    bool
	}{
		{
			name:       "Success: check overall health (default serving)",
			setup:      func(s *Server) {},
			service:    "",
			wantStatus: Serving,
		},
		{
			name: "Success: check specific service",
			setup: func(s *Server) {
				s.SetServingStatus("myservice", NotServing)
			},
			service:    "myservice",
			wantStatus: NotServing,
		},
		{
			name:       "Success: check unknown service",
			setup:      func(s *Server) {},
			service:    "unknown",
			wantStatus: ServiceUnknown,
		},
	}

	for _, test := range tests {
		srv := NewServer()
		test.setup(srv)

		// Create request
		req := NewHealthCheckRequest(ctx).SetService(test.service)
		reqBytes, err := req.Marshal()
		if err != nil {
			t.Errorf("[TestServerCheck](%s): failed to marshal request: %v", test.name, err)
			continue
		}

		// Call Check
		respBytes, err := srv.Check(ctx, reqBytes, []msgs.Metadata{})
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestServerCheck](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestServerCheck](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		// Unmarshal response
		resp := NewHealthCheckResponse(ctx)
		if err := resp.Unmarshal(respBytes); err != nil {
			t.Errorf("[TestServerCheck](%s): failed to unmarshal response: %v", test.name, err)
			continue
		}

		if resp.Status() != test.wantStatus {
			t.Errorf("[TestServerCheck](%s): got status = %v, want %v", test.name, resp.Status(), test.wantStatus)
		}
	}
}

func TestServerHandler(t *testing.T) {
	srv := NewServer()
	handler := srv.Handler()
	if handler.HandleFunc == nil {
		t.Error("[TestServerHandler]: got HandleFunc == nil, want non-nil")
	}
}
