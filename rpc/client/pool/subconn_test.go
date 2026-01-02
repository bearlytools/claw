package pool

import (
	"testing"

	"github.com/bearlytools/claw/rpc/health"
	"github.com/bearlytools/claw/rpc/transport/resolver"
)

func TestConnStateString(t *testing.T) {
	tests := []struct {
		state ConnState
		want  string
	}{
		{StateIdle, "IDLE"},
		{StateConnecting, "CONNECTING"},
		{StateReady, "READY"},
		{StateTransientFailure, "TRANSIENT_FAILURE"},
		{StateShutdown, "SHUTDOWN"},
		{ConnState(99), "UNKNOWN"},
	}

	for _, test := range tests {
		if got := test.state.String(); got != test.want {
			t.Errorf("[TestConnStateString]: %d.String() = %q, want %q", test.state, got, test.want)
		}
	}
}

func TestSubConnNewAndState(t *testing.T) {
	addr := resolver.Address{Addr: "localhost:8080", Weight: 10}
	sc := newSubConn(addr, nil, nil)

	if sc.State() != StateIdle {
		t.Errorf("[TestSubConnNewAndState]: initial state = %v, want %v", sc.State(), StateIdle)
	}

	if sc.HealthStatus() != health.Unknown {
		t.Errorf("[TestSubConnNewAndState]: initial health = %v, want %v", sc.HealthStatus(), health.Unknown)
	}

	if sc.Addr().Addr != addr.Addr {
		t.Errorf("[TestSubConnNewAndState]: addr = %q, want %q", sc.Addr().Addr, addr.Addr)
	}

	if sc.IsReady() {
		t.Error("[TestSubConnNewAndState]: IsReady() = true, want false for new SubConn")
	}

	if sc.LastError() != nil {
		t.Errorf("[TestSubConnNewAndState]: LastError() = %v, want nil", sc.LastError())
	}

	if sc.Conn() != nil {
		t.Error("[TestSubConnNewAndState]: Conn() should be nil for unconnected SubConn")
	}
}

func TestSubConnIsReady(t *testing.T) {
	addr := resolver.Address{Addr: "localhost:8080"}
	sc := newSubConn(addr, nil, nil)

	// Not ready initially (idle state)
	if sc.IsReady() {
		t.Error("[TestSubConnIsReady]: IsReady() = true for idle SubConn")
	}

	// Set to ready state but not healthy
	sc.mu.Lock()
	sc.state = StateReady
	sc.health = health.NotServing
	sc.mu.Unlock()

	if sc.IsReady() {
		t.Error("[TestSubConnIsReady]: IsReady() = true for ready but unhealthy SubConn")
	}

	// Set to ready and healthy
	sc.mu.Lock()
	sc.health = health.Serving
	sc.mu.Unlock()

	if !sc.IsReady() {
		t.Error("[TestSubConnIsReady]: IsReady() = false for ready and healthy SubConn")
	}

	// Set to transient failure
	sc.mu.Lock()
	sc.state = StateTransientFailure
	sc.mu.Unlock()

	if sc.IsReady() {
		t.Error("[TestSubConnIsReady]: IsReady() = true for transient failure SubConn")
	}
}

func TestSubConnSetHealth(t *testing.T) {
	addr := resolver.Address{Addr: "localhost:8080"}
	sc := newSubConn(addr, nil, nil)

	if sc.HealthStatus() != health.Unknown {
		t.Errorf("[TestSubConnSetHealth]: initial health = %v, want %v", sc.HealthStatus(), health.Unknown)
	}

	sc.setHealth(health.Serving)
	if sc.HealthStatus() != health.Serving {
		t.Errorf("[TestSubConnSetHealth]: health after setHealth(Serving) = %v, want %v", sc.HealthStatus(), health.Serving)
	}

	sc.setHealth(health.NotServing)
	if sc.HealthStatus() != health.NotServing {
		t.Errorf("[TestSubConnSetHealth]: health after setHealth(NotServing) = %v, want %v", sc.HealthStatus(), health.NotServing)
	}
}

func TestSubConnDisconnect(t *testing.T) {
	addr := resolver.Address{Addr: "localhost:8080"}
	sc := newSubConn(addr, nil, nil)

	// Simulate connected state
	sc.mu.Lock()
	sc.state = StateReady
	sc.health = health.Serving
	sc.mu.Unlock()

	sc.disconnect()

	if sc.State() != StateIdle {
		t.Errorf("[TestSubConnDisconnect]: state after disconnect = %v, want %v", sc.State(), StateIdle)
	}

	if sc.HealthStatus() != health.Unknown {
		t.Errorf("[TestSubConnDisconnect]: health after disconnect = %v, want %v", sc.HealthStatus(), health.Unknown)
	}
}

func TestSubConnShutdown(t *testing.T) {
	addr := resolver.Address{Addr: "localhost:8080"}
	sc := newSubConn(addr, nil, nil)

	// Simulate connected state
	sc.mu.Lock()
	sc.state = StateReady
	sc.mu.Unlock()

	sc.shutdown()

	if sc.State() != StateShutdown {
		t.Errorf("[TestSubConnShutdown]: state after shutdown = %v, want %v", sc.State(), StateShutdown)
	}

	// Calling shutdown again should be a no-op
	sc.shutdown()
	if sc.State() != StateShutdown {
		t.Errorf("[TestSubConnShutdown]: state after second shutdown = %v, want %v", sc.State(), StateShutdown)
	}

	// closeCh should be closed
	select {
	case <-sc.closeCh:
		// Expected
	default:
		t.Error("[TestSubConnShutdown]: closeCh should be closed after shutdown")
	}
}

func TestSubConnShutdownDisconnected(t *testing.T) {
	addr := resolver.Address{Addr: "localhost:8080"}
	sc := newSubConn(addr, nil, nil)

	// Shutdown while idle should work
	sc.shutdown()

	if sc.State() != StateShutdown {
		t.Errorf("[TestSubConnShutdownDisconnected]: state after shutdown = %v, want %v", sc.State(), StateShutdown)
	}
}
