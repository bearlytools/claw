package server

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/client"
	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// pipe creates a connected pair of net.Conn for testing.
func pipe() (io.ReadWriteCloser, io.ReadWriteCloser) {
	return net.Pipe()
}

func TestShutdownNoConnections(t *testing.T) {
	ctx := t.Context()
	srv := New()
	srv.Register(ctx, "test", "TestService", "Echo", SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	// Shutdown with no connections should complete immediately.
	shutdownCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := srv.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("TestShutdownNoConnections: Shutdown returned error: %v", err)
	}
}

func TestShutdownWaitsForHandlers(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that blocks until signaled.
	handlerBlocking := make(chan struct{})
	handlerDone := make(chan struct{})

	srv := New()
	srv.Register(ctx, "test", "TestService", "SlowEcho", SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			close(handlerBlocking)
			<-handlerDone
			return req, nil
		},
	})

	// Start server in background.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Create client and start a call.
	conn := client.New(ctx, clientConn)
	syncClient, err := conn.Sync(ctx, "test", "TestService", "SlowEcho")
	if err != nil {
		t.Fatalf("TestShutdownWaitsForHandlers: failed to create sync client: %v", err)
	}

	// Start the RPC in a goroutine.
	callDone := make(chan struct{})
	go func() {
		syncClient.Call(ctx, []byte("test"))
		close(callDone)
	}()

	// Wait for handler to be running.
	<-handlerBlocking

	// Start shutdown - it should wait for the handler.
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		shutdownDone <- srv.Shutdown(shutdownCtx)
	}()

	// Shutdown should be waiting (not done yet).
	select {
	case <-shutdownDone:
		t.Errorf("TestShutdownWaitsForHandlers: Shutdown returned before handler completed")
	case <-time.After(50 * time.Millisecond):
		// Expected - shutdown is still waiting.
	}

	// Now allow the handler to complete.
	close(handlerDone)

	// Wait for the RPC call to complete.
	<-callDone
	syncClient.Close()
	conn.Close()

	// Shutdown should now complete.
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("TestShutdownWaitsForHandlers: Shutdown returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("TestShutdownWaitsForHandlers: Shutdown did not complete in time")
	}

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestShutdownWaitsForHandlers: server did not shut down in time")
	}
}

func TestShutdownTimeout(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that blocks forever.
	handlerBlocking := make(chan struct{})

	srv := New()
	srv.Register(ctx, "test", "TestService", "BlockingEcho", SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			close(handlerBlocking)
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})

	// Start server in background.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Create client and start a call.
	conn := client.New(ctx, clientConn)
	syncClient, err := conn.Sync(ctx, "test", "TestService", "BlockingEcho")
	if err != nil {
		t.Fatalf("TestShutdownTimeout: failed to create sync client: %v", err)
	}

	// Start the RPC in a goroutine (will block forever).
	go func() {
		syncClient.Call(ctx, []byte("test"))
	}()

	// Wait for handler to be running.
	<-handlerBlocking

	// Shutdown with short timeout should time out.
	shutdownCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	err = srv.Shutdown(shutdownCtx)
	if err == nil {
		t.Errorf("TestShutdownTimeout: expected timeout error, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("TestShutdownTimeout: expected DeadlineExceeded, got %v", err)
	}

	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestShutdownTimeout: server did not shut down in time")
	}
}

func TestNewSessionsDuringServerDraining(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that blocks until signaled.
	handlerBlocking := make(chan struct{})
	handlerDone := make(chan struct{})

	srv := New()
	srv.Register(ctx, "test", "TestService", "SlowEcho", SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			select {
			case <-handlerBlocking:
			default:
				close(handlerBlocking)
			}
			<-handlerDone
			return req, nil
		},
	})

	// Start server in background.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Create client and start a call.
	conn := client.New(ctx, clientConn)
	syncClient, err := conn.Sync(ctx, "test", "TestService", "SlowEcho")
	if err != nil {
		t.Fatalf("TestNewSessionsDuringServerDraining: failed to create sync client: %v", err)
	}

	// Start the RPC in a goroutine.
	go func() {
		syncClient.Call(ctx, []byte("test"))
	}()

	// Wait for handler to be running.
	<-handlerBlocking

	// Start shutdown in background.
	go func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	// Give time for draining to start.
	time.Sleep(50 * time.Millisecond)

	// Server should be draining now.
	if !srv.IsDraining() {
		t.Errorf("TestNewSessionsDuringServerDraining: expected IsDraining() to return true")
	}

	// Try to create a new session - should fail with unavailable error.
	_, err = conn.Sync(ctx, "test", "TestService", "SlowEcho")
	if err == nil {
		t.Errorf("TestNewSessionsDuringServerDraining: expected error when creating session during draining")
	}

	// Allow the handler to complete.
	close(handlerDone)
	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestNewSessionsDuringServerDraining: server did not shut down in time")
	}
}

func TestServerIsDraining(t *testing.T) {
	srv := New()

	// Initially not draining.
	if srv.IsDraining() {
		t.Errorf("TestServerIsDraining: expected IsDraining() to return false initially")
	}
}

func TestServerConnGracefulClose(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that blocks until signaled.
	handlerBlocking := make(chan struct{})
	handlerDone := make(chan struct{})

	srv := New()
	srv.Register(ctx, "test", "TestService", "SlowEcho", SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			close(handlerBlocking)
			<-handlerDone
			return req, nil
		},
	})

	// Start server in background.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Create client and start a call.
	conn := client.New(ctx, clientConn)
	syncClient, err := conn.Sync(ctx, "test", "TestService", "SlowEcho")
	if err != nil {
		t.Fatalf("TestServerConnGracefulClose: failed to create sync client: %v", err)
	}

	// Start the RPC in a goroutine.
	callDone := make(chan struct{})
	go func() {
		syncClient.Call(ctx, []byte("test"))
		close(callDone)
	}()

	// Wait for handler to be running.
	<-handlerBlocking

	// Allow the handler to complete.
	close(handlerDone)

	// Wait for the RPC call to complete.
	<-callDone

	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestServerConnGracefulClose: server did not shut down in time")
	}
}

func TestPackingSyncRPC(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	srv := New(WithPacking(true))
	srv.Register(ctx, "test", "TestService", "Echo", SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	// Start server in background.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Create client with packing enabled.
	conn := client.New(ctx, clientConn, client.WithPacking(true))
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestPackingSyncRPC: failed to create sync client: %v", err)
	}

	// Test with single request.
	resp, err := syncClient.Call(ctx, []byte("hello packed world"))
	if err != nil {
		t.Fatalf("TestPackingSyncRPC: Call returned error: %v", err)
	}
	if string(resp) != "hello packed world" {
		t.Errorf("TestPackingSyncRPC: expected 'hello packed world', got '%s'", string(resp))
	}

	// Test with multiple requests to ensure packing continues to work.
	for i := 0; i < 10; i++ {
		resp, err = syncClient.Call(ctx, []byte("packed message"))
		if err != nil {
			t.Fatalf("TestPackingSyncRPC: Call %d returned error: %v", i, err)
		}
		if string(resp) != "packed message" {
			t.Errorf("TestPackingSyncRPC: Call %d: expected 'packed message', got '%s'", i, string(resp))
		}
	}

	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestPackingSyncRPC: server did not shut down in time")
	}
}

func TestPackingLargePayload(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	srv := New(WithPacking(true))
	srv.Register(ctx, "test", "TestService", "Echo", SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	// Start server in background.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Create client with packing enabled.
	conn := client.New(ctx, clientConn, client.WithPacking(true))
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestPackingLargePayload: failed to create sync client: %v", err)
	}

	// Test with a large payload containing zeros (should compress well).
	largePayload := make([]byte, 64*1024) // 64 KB
	for i := 0; i < len(largePayload); i += 8 {
		largePayload[i] = byte(i % 256)
	}

	resp, err := syncClient.Call(ctx, largePayload)
	if err != nil {
		t.Fatalf("TestPackingLargePayload: Call returned error: %v", err)
	}
	if len(resp) != len(largePayload) {
		t.Errorf("TestPackingLargePayload: expected length %d, got %d", len(largePayload), len(resp))
	}
	for i := range resp {
		if resp[i] != largePayload[i] {
			t.Errorf("TestPackingLargePayload: mismatch at byte %d: expected %d, got %d", i, largePayload[i], resp[i])
			break
		}
	}

	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestPackingLargePayload: server did not shut down in time")
	}
}

func TestPackingServerDisabledClientEnabled(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server has packing disabled.
	srv := New()
	srv.Register(ctx, "test", "TestService", "Echo", SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	// Start server in background.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Client requests packing but server doesn't support it.
	conn := client.New(ctx, clientConn, client.WithPacking(true))
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestPackingServerDisabledClientEnabled: failed to create sync client: %v", err)
	}

	// Should still work, just without packing.
	resp, err := syncClient.Call(ctx, []byte("hello"))
	if err != nil {
		t.Fatalf("TestPackingServerDisabledClientEnabled: Call returned error: %v", err)
	}
	if string(resp) != "hello" {
		t.Errorf("TestPackingServerDisabledClientEnabled: expected 'hello', got '%s'", string(resp))
	}

	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestPackingServerDisabledClientEnabled: server did not shut down in time")
	}
}
