package pool

import (
	"testing"
	"time"

	"github.com/bearlytools/claw/rpc/client"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.balancer == nil {
		t.Error("[TestDefaultConfig]: balancer should not be nil")
	}
	if _, ok := cfg.balancer.(*RoundRobinBalancer); !ok {
		t.Error("[TestDefaultConfig]: default balancer should be RoundRobinBalancer")
	}

	if cfg.healthCheckInterval != 30*time.Second {
		t.Errorf("[TestDefaultConfig]: healthCheckInterval = %v, want %v", cfg.healthCheckInterval, 30*time.Second)
	}

	if cfg.healthCheckTimeout != 5*time.Second {
		t.Errorf("[TestDefaultConfig]: healthCheckTimeout = %v, want %v", cfg.healthCheckTimeout, 5*time.Second)
	}

	if cfg.minConns != 1 {
		t.Errorf("[TestDefaultConfig]: minConns = %d, want 1", cfg.minConns)
	}

	if !cfg.enableHealthCheck {
		t.Error("[TestDefaultConfig]: enableHealthCheck should be true by default")
	}
}

func TestWithBalancer(t *testing.T) {
	cfg := defaultConfig()

	newBalancer := &PickFirstBalancer{}
	WithBalancer(newBalancer)(cfg)

	if cfg.balancer != newBalancer {
		t.Error("[TestWithBalancer]: balancer was not set")
	}

	// Nil balancer should not change the config
	originalBalancer := cfg.balancer
	WithBalancer(nil)(cfg)
	if cfg.balancer != originalBalancer {
		t.Error("[TestWithBalancer]: nil balancer should not change config")
	}
}

func TestWithHealthCheckInterval(t *testing.T) {
	cfg := defaultConfig()

	WithHealthCheckInterval(10 * time.Second)(cfg)
	if cfg.healthCheckInterval != 10*time.Second {
		t.Errorf("[TestWithHealthCheckInterval]: healthCheckInterval = %v, want %v", cfg.healthCheckInterval, 10*time.Second)
	}
	if !cfg.enableHealthCheck {
		t.Error("[TestWithHealthCheckInterval]: enableHealthCheck should be true for positive interval")
	}

	// Zero disables health checking
	WithHealthCheckInterval(0)(cfg)
	if cfg.healthCheckInterval != 0 {
		t.Errorf("[TestWithHealthCheckInterval]: healthCheckInterval = %v, want 0", cfg.healthCheckInterval)
	}
	if cfg.enableHealthCheck {
		t.Error("[TestWithHealthCheckInterval]: enableHealthCheck should be false for zero interval")
	}
}

func TestWithHealthCheckTimeout(t *testing.T) {
	cfg := defaultConfig()

	WithHealthCheckTimeout(3 * time.Second)(cfg)
	if cfg.healthCheckTimeout != 3*time.Second {
		t.Errorf("[TestWithHealthCheckTimeout]: healthCheckTimeout = %v, want %v", cfg.healthCheckTimeout, 3*time.Second)
	}
}

func TestWithClientOptions(t *testing.T) {
	cfg := defaultConfig()

	if len(cfg.clientOpts) != 0 {
		t.Error("[TestWithClientOptions]: clientOpts should be empty initially")
	}

	opt1 := client.WithPingInterval(10 * time.Second)
	WithClientOptions(opt1)(cfg)
	if len(cfg.clientOpts) != 1 {
		t.Errorf("[TestWithClientOptions]: len(clientOpts) = %d, want 1", len(cfg.clientOpts))
	}

	opt2 := client.WithPingTimeout(5 * time.Second)
	WithClientOptions(opt2)(cfg)
	if len(cfg.clientOpts) != 2 {
		t.Errorf("[TestWithClientOptions]: len(clientOpts) = %d, want 2", len(cfg.clientOpts))
	}
}

func TestWithMinConnections(t *testing.T) {
	cfg := defaultConfig()

	WithMinConnections(5)(cfg)
	if cfg.minConns != 5 {
		t.Errorf("[TestWithMinConnections]: minConns = %d, want 5", cfg.minConns)
	}

	// Zero or negative should not change config
	WithMinConnections(0)(cfg)
	if cfg.minConns != 5 {
		t.Errorf("[TestWithMinConnections]: minConns = %d after zero, want 5 (unchanged)", cfg.minConns)
	}

	WithMinConnections(-1)(cfg)
	if cfg.minConns != 5 {
		t.Errorf("[TestWithMinConnections]: minConns = %d after negative, want 5 (unchanged)", cfg.minConns)
	}
}

func TestWithHealthCheckDisabled(t *testing.T) {
	cfg := defaultConfig()

	if !cfg.enableHealthCheck {
		t.Error("[TestWithHealthCheckDisabled]: enableHealthCheck should be true initially")
	}

	WithHealthCheckDisabled()(cfg)
	if cfg.enableHealthCheck {
		t.Error("[TestWithHealthCheckDisabled]: enableHealthCheck should be false after disabling")
	}
}
