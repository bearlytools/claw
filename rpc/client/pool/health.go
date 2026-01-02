package pool

import (
	"time"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/health"
)

// startHealthChecker starts the background health checking goroutine.
// It periodically checks the health of all SubConns and updates the ready list.
func (p *Pool) startHealthChecker(ctx context.Context) {
	if !p.cfg.enableHealthCheck || p.cfg.healthCheckInterval <= 0 {
		return
	}

	pool := context.Pool(ctx)
	pool.Submit(ctx, func() {
		p.healthCheckLoop(ctx)
	})
}

// healthCheckLoop runs the periodic health check loop.
func (p *Pool) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(p.cfg.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.closed:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.checkAllHealth(ctx)
		}
	}
}

// checkAllHealth checks the health of all SubConns.
func (p *Pool) checkAllHealth(ctx context.Context) {
	p.mu.Lock()
	subConns := make([]*SubConn, 0, len(p.subConns))
	for _, sc := range p.subConns {
		subConns = append(subConns, sc)
	}
	p.mu.Unlock()

	// Check each SubConn's health with timeout
	for _, sc := range subConns {
		select {
		case <-p.closed:
			return
		case <-ctx.Done():
			return
		default:
		}

		if sc.State() != StateReady {
			continue
		}

		checkCtx, cancel := context.WithTimeout(ctx, p.cfg.healthCheckTimeout)
		status := sc.CheckHealth(checkCtx)
		cancel()

		// If health check failed, the SubConn will handle it internally
		if status != health.Serving {
			p.handleUnhealthySubConn(ctx, sc)
		}
	}

	// Update ready list after health checks
	p.updateReadySubConns()
}

// handleUnhealthySubConn handles a SubConn that failed health check.
func (p *Pool) handleUnhealthySubConn(ctx context.Context, sc *SubConn) {
	// For now, just log or mark. The SubConn will be excluded from ready list.
	// Future: could implement circuit breaker logic here.
	sc.setHealth(health.NotServing)
}

// updateReadySubConns rebuilds the ready SubConn list.
// If new SubConns become ready, it broadcasts to any waiting goroutines.
func (p *Pool) updateReadySubConns() {
	p.mu.Lock()

	prevCount := len(p.readySubConns)

	ready := make([]*SubConn, 0, len(p.subConns))
	for _, sc := range p.subConns {
		if sc.IsReady() {
			ready = append(ready, sc)
		}
	}
	p.readySubConns = ready

	// If we now have ready SubConns and previously had none (or fewer),
	// broadcast to wake up any waiting goroutines.
	if len(ready) > 0 && prevCount == 0 {
		close(p.readyBroadcast)
		p.readyBroadcast = make(chan struct{})
	}

	p.mu.Unlock()
}

// maintainMinConns ensures at least minConns SubConns are connected.
// This is called periodically or when SubConns fail.
func (p *Pool) maintainMinConns(ctx context.Context) {
	p.mu.Lock()
	readyCount := len(p.readySubConns)
	minConns := p.cfg.minConns
	subConns := make([]*SubConn, 0, len(p.subConns))
	for _, sc := range p.subConns {
		subConns = append(subConns, sc)
	}
	p.mu.Unlock()

	if readyCount >= minConns {
		return
	}

	// Try to connect idle or failed SubConns
	for _, sc := range subConns {
		if readyCount >= minConns {
			break
		}

		state := sc.State()
		if state == StateIdle || state == StateTransientFailure {
			sc.Connect(ctx)
			readyCount++ // Optimistic count
		}
	}
}
