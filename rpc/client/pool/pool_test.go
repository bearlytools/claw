package pool

import (
	"testing"
	"time"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/client"
	"github.com/bearlytools/claw/rpc/health"
	"github.com/bearlytools/claw/rpc/transport/resolver"
)

func TestGetSubConnWaitNoReadyFastFail(t *testing.T) {
	ctx := t.Context()

	p := &Pool{
		subConns:       make(map[string]*SubConn),
		readySubConns:  nil,
		readyBroadcast: make(chan struct{}),
		closed:         make(chan struct{}),
		cfg:            defaultConfig(),
	}

	// Without wait-for-ready, should fail immediately
	_, err := p.getSubConnWait(ctx, false)
	if err != ErrNoReadySubConns {
		t.Errorf("TestGetSubConnWaitNoReadyFastFail: got err=%v, want %v", err, ErrNoReadySubConns)
	}
}

func TestGetSubConnWaitContextCancelled(t *testing.T) {
	ctx := t.Context()
	ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	p := &Pool{
		subConns:       make(map[string]*SubConn),
		readySubConns:  nil,
		readyBroadcast: make(chan struct{}),
		closed:         make(chan struct{}),
		cfg:            defaultConfig(),
	}

	// With wait-for-ready, should wait until context is done
	start := time.Now()
	_, err := p.getSubConnWait(ctx, true)
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("TestGetSubConnWaitContextCancelled: got err=%v, want context.DeadlineExceeded", err)
	}

	// Should have waited for timeout
	if elapsed < 40*time.Millisecond {
		t.Errorf("TestGetSubConnWaitContextCancelled: elapsed=%v, should have waited for timeout", elapsed)
	}
}

func TestGetSubConnWaitPoolClosed(t *testing.T) {
	ctx := t.Context()

	p := &Pool{
		subConns:       make(map[string]*SubConn),
		readySubConns:  nil,
		readyBroadcast: make(chan struct{}),
		closed:         make(chan struct{}),
		cfg:            defaultConfig(),
	}

	close(p.closed)

	_, err := p.getSubConnWait(ctx, true)
	if err != ErrPoolClosed {
		t.Errorf("TestGetSubConnWaitPoolClosed: got err=%v, want %v", err, ErrPoolClosed)
	}
}

func TestGetSubConnWaitReturnsReady(t *testing.T) {
	ctx := t.Context()

	sc := newSubConn(resolver.Address{Addr: "localhost:8080"}, nil, nil)
	sc.mu.Lock()
	sc.state = StateReady
	sc.health = health.Serving
	sc.mu.Unlock()

	p := &Pool{
		subConns:       map[string]*SubConn{"localhost:8080": sc},
		readySubConns:  []*SubConn{sc},
		readyBroadcast: make(chan struct{}),
		closed:         make(chan struct{}),
		cfg:            defaultConfig(),
	}

	got, err := p.getSubConnWait(ctx, true)
	if err != nil {
		t.Errorf("TestGetSubConnWaitReturnsReady: unexpected error: %v", err)
	}
	if got != sc {
		t.Errorf("TestGetSubConnWaitReturnsReady: got wrong SubConn")
	}
}

func TestGetSubConnWaitBlocksUntilReady(t *testing.T) {
	ctx := t.Context()

	sc := newSubConn(resolver.Address{Addr: "localhost:8080"}, nil, nil)

	p := &Pool{
		subConns:       map[string]*SubConn{"localhost:8080": sc},
		readySubConns:  nil,
		readyBroadcast: make(chan struct{}),
		closed:         make(chan struct{}),
		cfg:            defaultConfig(),
	}

	done := make(chan struct{})
	var gotErr error
	var gotSC *SubConn

	go func() {
		gotSC, gotErr = p.getSubConnWait(ctx, true)
		close(done)
	}()

	// Give the goroutine time to start waiting
	time.Sleep(20 * time.Millisecond)

	// Make SubConn ready
	sc.mu.Lock()
	sc.state = StateReady
	sc.health = health.Serving
	sc.mu.Unlock()

	// Update ready list and broadcast
	p.updateReadySubConns()

	// Wait for getSubConnWait to return
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("TestGetSubConnWaitBlocksUntilReady: timed out waiting for getSubConnWait to return")
	}

	if gotErr != nil {
		t.Errorf("TestGetSubConnWaitBlocksUntilReady: unexpected error: %v", gotErr)
	}
	if gotSC != sc {
		t.Errorf("TestGetSubConnWaitBlocksUntilReady: got wrong SubConn")
	}
}

func TestUpdateReadySubConnsBroadcasts(t *testing.T) {
	sc := newSubConn(resolver.Address{Addr: "localhost:8080"}, nil, nil)

	p := &Pool{
		subConns:       map[string]*SubConn{"localhost:8080": sc},
		readySubConns:  nil,
		readyBroadcast: make(chan struct{}),
		closed:         make(chan struct{}),
		cfg:            defaultConfig(),
	}

	// Store the broadcast channel before update
	broadcast := p.readyBroadcast

	// Make SubConn ready
	sc.mu.Lock()
	sc.state = StateReady
	sc.health = health.Serving
	sc.mu.Unlock()

	// Update should broadcast
	p.updateReadySubConns()

	// Old broadcast channel should be closed
	select {
	case <-broadcast:
		// Expected - channel was closed
	default:
		t.Error("TestUpdateReadySubConnsBroadcasts: broadcast channel was not closed")
	}

	// New broadcast channel should be open
	p.mu.Lock()
	newBroadcast := p.readyBroadcast
	p.mu.Unlock()

	select {
	case <-newBroadcast:
		t.Error("TestUpdateReadySubConnsBroadcasts: new broadcast channel should be open")
	default:
		// Expected
	}
}

func TestGetWaitForReady(t *testing.T) {
	tests := []struct {
		name string
		opts []client.CallOption
		want bool
	}{
		{
			name: "Success: no options returns false",
			opts: nil,
			want: false,
		},
		{
			name: "Success: WithWaitForReady(true) returns true",
			opts: []client.CallOption{client.WithWaitForReady(true)},
			want: true,
		},
		{
			name: "Success: WithWaitForReady(false) returns false",
			opts: []client.CallOption{client.WithWaitForReady(false)},
			want: false,
		},
		{
			name: "Success: last option wins",
			opts: []client.CallOption{client.WithWaitForReady(true), client.WithWaitForReady(false)},
			want: false,
		},
	}

	for _, test := range tests {
		got := client.GetWaitForReady(test.opts...)
		if got != test.want {
			t.Errorf("TestGetWaitForReady(%s): got %v, want %v", test.name, got, test.want)
		}
	}
}
