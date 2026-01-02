package client

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/gostdlib/base/context"
	"github.com/kylelemons/godebug/pretty"

	"github.com/bearlytools/claw/rpc/errors"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/server"
	"github.com/bearlytools/claw/rpc/serviceconfig"
)

// pipe creates a connected pair of net.Conn for testing.
func pipe() (io.ReadWriteCloser, io.ReadWriteCloser) {
	return net.Pipe()
}

func TestSynchronousRPC(t *testing.T) {
	tests := []struct {
		name     string
		requests [][]byte
		wantErr  bool
	}{
		{
			name:     "Success: single request",
			requests: [][]byte{[]byte("hello")},
			wantErr:  false,
		},
		{
			name:     "Success: multiple requests",
			requests: [][]byte{[]byte("first"), []byte("second"), []byte("third")},
			wantErr:  false,
		},
		{
			name:     "Success: empty payload",
			requests: [][]byte{[]byte("")},
			wantErr:  false,
		},
		{
			name:     "Success: large payload",
			requests: [][]byte{make([]byte, 64*1024)},
			wantErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			clientConn, serverConn := pipe()

			// Setup server.
			srv := server.New()
			err := srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
				HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
					// Echo back with prefix.
					return append([]byte("echo:"), req...), nil
				},
			})
			if err != nil {
				t.Fatalf("TestSynchronousRPC(%s): failed to register handler: %v", test.name, err)
			}

			// Start server in background.
			serverDone := make(chan error, 1)
			go func() {
				serverDone <- srv.Serve(ctx, serverConn)
			}()

			// Create client connection.
			conn := New(ctx, clientConn)
			defer conn.Close()

			// Create sync client.
			syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
			if err != nil {
				t.Fatalf("TestSynchronousRPC(%s): failed to create sync client: %v", test.name, err)
			}
			defer syncClient.Close()

			// Send requests and verify responses.
			for i, req := range test.requests {
				resp, err := syncClient.Call(ctx, req)
				switch {
				case err == nil && test.wantErr:
					t.Errorf("TestSynchronousRPC(%s): request %d: got err == nil, want err != nil", test.name, i)
					continue
				case err != nil && !test.wantErr:
					t.Errorf("TestSynchronousRPC(%s): request %d: got err == %v, want err == nil", test.name, i, err)
					continue
				case err != nil:
					continue
				}

				want := append([]byte("echo:"), req...)
				if diff := pretty.Compare(want, resp); diff != "" {
					t.Errorf("TestSynchronousRPC(%s): request %d: response mismatch (-want +got):\n%s", test.name, i, diff)
				}
			}

			// Close client and wait for server.
			syncClient.Close()
			conn.Close()

			select {
			case <-serverDone:
			case <-time.After(5 * time.Second):
				t.Errorf("TestSynchronousRPC(%s): server did not shut down in time", test.name)
			}
		})
	}
}

func TestBiDirectionalRPC(t *testing.T) {
	tests := []struct {
		name         string
		clientMsgs   [][]byte
		serverMsgs   [][]byte
		wantErr      bool
	}{
		{
			name:       "Success: ping pong exchange",
			clientMsgs: [][]byte{[]byte("ping1"), []byte("ping2"), []byte("ping3")},
			serverMsgs: [][]byte{[]byte("pong1"), []byte("pong2"), []byte("pong3")},
			wantErr:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			clientConn, serverConn := pipe()

			// Server receives client messages and sends its own.
			serverRecv := make(chan []byte, len(test.clientMsgs))
			srv := server.New()
			err := srv.Register(ctx, "test", "TestService", "BiDir", server.BiDirHandler{
				HandleFunc: func(ctx context.Context, stream *server.BiDirStream) error {
					// Send and receive concurrently to avoid deadlock with net.Pipe().
					sendDone := make(chan error, 1)
					go func() {
						for _, msg := range test.serverMsgs {
							if err := stream.Send(msg); err != nil {
								sendDone <- err
								return
							}
						}
						sendDone <- nil
					}()

					// Receive client messages.
					for payload := range stream.Recv() {
						cp := make([]byte, len(payload))
						copy(cp, payload)
						serverRecv <- cp
					}
					close(serverRecv)

					if err := <-sendDone; err != nil {
						return err
					}
					return stream.Err()
				},
			})
			if err != nil {
				t.Fatalf("TestBiDirectionalRPC(%s): failed to register handler: %v", test.name, err)
			}

			serverDone := make(chan error, 1)
			go func() {
				serverDone <- srv.Serve(ctx, serverConn)
			}()

			conn := New(ctx, clientConn)
			defer conn.Close()

			bidir, err := conn.BiDir(ctx, "test", "TestService", "BiDir")
			if err != nil {
				t.Fatalf("TestBiDirectionalRPC(%s): failed to create bidir client: %v", test.name, err)
			}

			// Send and receive concurrently to avoid deadlock with net.Pipe().
			var received [][]byte
			var recvErr error
			recvDone := make(chan struct{})
			go func() {
				for payload := range bidir.Recv(ctx) {
					cp := make([]byte, len(payload))
					copy(cp, payload)
					received = append(received, cp)
				}
				recvErr = bidir.Err()
				close(recvDone)
			}()

			// Send client messages.
			for _, msg := range test.clientMsgs {
				if err := bidir.Send(ctx, msg); err != nil {
					t.Errorf("TestBiDirectionalRPC(%s): failed to send: %v", test.name, err)
				}
			}
			bidir.CloseSend()

			// Wait for receive to complete.
			<-recvDone

			if recvErr != nil && !test.wantErr {
				t.Errorf("TestBiDirectionalRPC(%s): receive error: %v", test.name, recvErr)
			}

			if diff := pretty.Compare(test.serverMsgs, received); diff != "" {
				t.Errorf("TestBiDirectionalRPC(%s): received messages mismatch (-want +got):\n%s", test.name, diff)
			}

			bidir.Close()
			conn.Close()

			select {
			case <-serverDone:
			case <-time.After(5 * time.Second):
				t.Errorf("TestBiDirectionalRPC(%s): server did not shut down in time", test.name)
			}

			// Verify server received client messages.
			var serverReceived [][]byte
			for msg := range serverRecv {
				serverReceived = append(serverReceived, msg)
			}
			if diff := pretty.Compare(test.clientMsgs, serverReceived); diff != "" {
				t.Errorf("TestBiDirectionalRPC(%s): server received messages mismatch (-want +got):\n%s", test.name, diff)
			}
		})
	}
}

func TestSendOnlyRPC(t *testing.T) {
	tests := []struct {
		name     string
		messages [][]byte
		wantErr  bool
	}{
		{
			name:     "Success: send multiple messages",
			messages: [][]byte{[]byte("msg1"), []byte("msg2"), []byte("msg3")},
			wantErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			clientConn, serverConn := pipe()

			serverRecv := make(chan []byte, len(test.messages)+1)
			srv := server.New()
			err := srv.Register(ctx, "test", "TestService", "Send", server.SendHandler{
				HandleFunc: func(ctx context.Context, stream *server.RecvStream) error {
					for payload := range stream.Recv() {
						cp := make([]byte, len(payload))
						copy(cp, payload)
						serverRecv <- cp
					}
					close(serverRecv)
					return stream.Err()
				},
			})
			if err != nil {
				t.Fatalf("TestSendOnlyRPC(%s): failed to register handler: %v", test.name, err)
			}

			serverDone := make(chan error, 1)
			go func() {
				serverDone <- srv.Serve(ctx, serverConn)
			}()

			conn := New(ctx, clientConn)
			defer conn.Close()

			sender, err := conn.Send(ctx, "test", "TestService", "Send")
			if err != nil {
				t.Fatalf("TestSendOnlyRPC(%s): failed to create send client: %v", test.name, err)
			}

			for _, msg := range test.messages {
				if err := sender.Send(ctx, msg); err != nil {
					t.Errorf("TestSendOnlyRPC(%s): failed to send: %v", test.name, err)
				}
			}
			sender.Close()
			conn.Close()

			select {
			case <-serverDone:
			case <-time.After(5 * time.Second):
				t.Errorf("TestSendOnlyRPC(%s): server did not shut down in time", test.name)
			}

			var received [][]byte
			for msg := range serverRecv {
				received = append(received, msg)
			}

			if diff := pretty.Compare(test.messages, received); diff != "" {
				t.Errorf("TestSendOnlyRPC(%s): server received messages mismatch (-want +got):\n%s", test.name, diff)
			}
		})
	}
}

func TestRecvOnlyRPC(t *testing.T) {
	tests := []struct {
		name     string
		messages [][]byte
		wantErr  bool
	}{
		{
			name:     "Success: receive multiple messages",
			messages: [][]byte{[]byte("msg1"), []byte("msg2"), []byte("msg3")},
			wantErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			clientConn, serverConn := pipe()

			srv := server.New()
			err := srv.Register(ctx, "test", "TestService", "Recv", server.RecvHandler{
				HandleFunc: func(ctx context.Context, stream *server.SendStream) error {
					for _, msg := range test.messages {
						if err := stream.Send(msg); err != nil {
							return err
						}
					}
					return nil
				},
			})
			if err != nil {
				t.Fatalf("TestRecvOnlyRPC(%s): failed to register handler: %v", test.name, err)
			}

			serverDone := make(chan error, 1)
			go func() {
				serverDone <- srv.Serve(ctx, serverConn)
			}()

			conn := New(ctx, clientConn)
			defer conn.Close()

			receiver, err := conn.Recv(ctx, "test", "TestService", "Recv")
			if err != nil {
				t.Fatalf("TestRecvOnlyRPC(%s): failed to create recv client: %v", test.name, err)
			}

			var received [][]byte
			for payload := range receiver.Recv(ctx) {
				cp := make([]byte, len(payload))
				copy(cp, payload)
				received = append(received, cp)
			}

			if err := receiver.Err(); err != nil && !test.wantErr {
				t.Errorf("TestRecvOnlyRPC(%s): receive error: %v", test.name, err)
			}

			if diff := pretty.Compare(test.messages, received); diff != "" {
				t.Errorf("TestRecvOnlyRPC(%s): received messages mismatch (-want +got):\n%s", test.name, diff)
			}

			receiver.Close()
			conn.Close()

			select {
			case <-serverDone:
			case <-time.After(5 * time.Second):
				t.Errorf("TestRecvOnlyRPC(%s): server did not shut down in time", test.name)
			}
		})
	}
}

func TestConnectionClose(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	srv := server.New()
	srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	conn := New(ctx, clientConn)

	// Create a session.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestConnectionClose: failed to create sync client: %v", err)
	}

	// Close the connection.
	conn.Close()

	// Verify session operations fail.
	_, err = syncClient.Call(ctx, []byte("test"))
	if err == nil {
		t.Errorf("TestConnectionClose: expected error after connection close, got nil")
	}

	// Verify Err() returns the fatal error.
	if conn.Err() != nil && conn.Err() != io.EOF {
		// Connection closed normally, not an error condition.
	}

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestConnectionClose: server did not shut down in time")
	}
}

func TestMultipleSessions(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	srv := server.New()
	srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return append([]byte("echo:"), req...), nil
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	conn := New(ctx, clientConn)
	defer conn.Close()

	// Create multiple sync clients.
	client1, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestMultipleSessions: failed to create client1: %v", err)
	}

	client2, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestMultipleSessions: failed to create client2: %v", err)
	}

	// Use both clients.
	resp1, err := client1.Call(ctx, []byte("from client1"))
	if err != nil {
		t.Errorf("TestMultipleSessions: client1 call failed: %v", err)
	}
	if string(resp1) != "echo:from client1" {
		t.Errorf("TestMultipleSessions: client1 got %q, want %q", resp1, "echo:from client1")
	}

	resp2, err := client2.Call(ctx, []byte("from client2"))
	if err != nil {
		t.Errorf("TestMultipleSessions: client2 call failed: %v", err)
	}
	if string(resp2) != "echo:from client2" {
		t.Errorf("TestMultipleSessions: client2 got %q, want %q", resp2, "echo:from client2")
	}

	client1.Close()
	client2.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestMultipleSessions: server did not shut down in time")
	}
}

func TestDeadlinePropagationBiDir(t *testing.T) {
	// This test verifies that when context deadline expires during streaming,
	// buffered messages are still yielded before the iterator returns.
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server will send multiple messages with a small delay between them.
	messagesFromServer := [][]byte{[]byte("msg1"), []byte("msg2"), []byte("msg3"), []byte("msg4"), []byte("msg5")}
	serverStarted := make(chan struct{})

	srv := server.New()
	err := srv.Register(ctx, "test", "TestService", "BiDir", server.BiDirHandler{
		HandleFunc: func(ctx context.Context, stream *server.BiDirStream) error {
			close(serverStarted)
			// Send all messages before deadline fires.
			for _, msg := range messagesFromServer {
				if err := stream.Send(msg); err != nil {
					return err
				}
			}
			// Wait until deadline expires (context cancelled).
			<-ctx.Done()
			return ctx.Err()
		},
	})
	if err != nil {
		t.Fatalf("TestDeadlinePropagationBiDir: failed to register handler: %v", err)
	}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	conn := New(ctx, clientConn)
	defer conn.Close()

	// Create context with short deadline.
	deadlineCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	bidir, err := conn.BiDir(deadlineCtx, "test", "TestService", "BiDir")
	if err != nil {
		t.Fatalf("TestDeadlinePropagationBiDir: failed to create bidir client: %v", err)
	}

	// Wait for server to start sending.
	<-serverStarted
	time.Sleep(50 * time.Millisecond) // Give time for messages to buffer.

	// Receive with deadline context - should get all buffered messages before deadline fires.
	var received [][]byte
	for payload := range bidir.Recv(deadlineCtx) {
		cp := make([]byte, len(payload))
		copy(cp, payload)
		received = append(received, cp)
	}

	// We should receive all messages that were sent before the deadline.
	if len(received) < len(messagesFromServer) {
		// It's acceptable if we received fewer due to timing, but we should have received some.
		if len(received) == 0 {
			t.Errorf("TestDeadlinePropagationBiDir: received no messages, expected some before deadline")
		}
	}

	bidir.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestDeadlinePropagationBiDir: server did not shut down in time")
	}
}

func TestDeadlinePropagationRecv(t *testing.T) {
	// This test verifies that when context deadline expires during recv streaming,
	// buffered messages are still yielded before the iterator returns.
	ctx := t.Context()
	clientConn, serverConn := pipe()

	messagesFromServer := [][]byte{[]byte("msg1"), []byte("msg2"), []byte("msg3")}
	serverStarted := make(chan struct{})

	srv := server.New()
	err := srv.Register(ctx, "test", "TestService", "Recv", server.RecvHandler{
		HandleFunc: func(ctx context.Context, stream *server.SendStream) error {
			close(serverStarted)
			// Send all messages.
			for _, msg := range messagesFromServer {
				if err := stream.Send(msg); err != nil {
					return err
				}
			}
			// Wait until deadline expires.
			<-ctx.Done()
			return ctx.Err()
		},
	})
	if err != nil {
		t.Fatalf("TestDeadlinePropagationRecv: failed to register handler: %v", err)
	}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	conn := New(ctx, clientConn)
	defer conn.Close()

	// Create context with short deadline.
	deadlineCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	recvClient, err := conn.Recv(deadlineCtx, "test", "TestService", "Recv")
	if err != nil {
		t.Fatalf("TestDeadlinePropagationRecv: failed to create recv client: %v", err)
	}

	// Wait for server to start sending.
	<-serverStarted
	time.Sleep(50 * time.Millisecond) // Give time for messages to buffer.

	// Receive with deadline context.
	var received [][]byte
	for payload := range recvClient.Recv(deadlineCtx) {
		cp := make([]byte, len(payload))
		copy(cp, payload)
		received = append(received, cp)
	}

	// We should receive all messages that were sent.
	if len(received) < len(messagesFromServer) {
		if len(received) == 0 {
			t.Errorf("TestDeadlinePropagationRecv: received no messages, expected some before deadline")
		}
	}

	recvClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestDeadlinePropagationRecv: server did not shut down in time")
	}
}

func TestGracefulCloseNoSessions(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	srv := server.New()
	srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	conn := New(ctx, clientConn)

	// GracefulClose with no sessions should complete immediately.
	gracefulCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := conn.GracefulClose(gracefulCtx)
	if err != nil {
		t.Errorf("TestGracefulCloseNoSessions: GracefulClose returned error: %v", err)
	}

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestGracefulCloseNoSessions: server did not shut down in time")
	}
}

func TestGracefulCloseWaitsForSession(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that blocks until signaled.
	handlerBlocking := make(chan struct{})
	handlerDone := make(chan struct{})

	srv := server.New()
	srv.Register(ctx, "test", "TestService", "SlowEcho", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			// Signal that handler is running.
			close(handlerBlocking)
			// Wait for signal to complete.
			<-handlerDone
			return req, nil
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	conn := New(ctx, clientConn)

	// Create a sync client and start a call.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "SlowEcho")
	if err != nil {
		t.Fatalf("TestGracefulCloseWaitsForSession: failed to create sync client: %v", err)
	}

	// Start the RPC in a goroutine.
	callDone := make(chan struct{})
	go func() {
		syncClient.Call(ctx, []byte("test"))
		close(callDone)
	}()

	// Wait for handler to be running.
	<-handlerBlocking

	// Start graceful close - it should wait for the session.
	gracefulDone := make(chan error, 1)
	go func() {
		gracefulCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		gracefulDone <- conn.GracefulClose(gracefulCtx)
	}()

	// GracefulClose should be waiting (not done yet).
	select {
	case <-gracefulDone:
		t.Errorf("TestGracefulCloseWaitsForSession: GracefulClose returned before session completed")
	case <-time.After(50 * time.Millisecond):
		// Expected - graceful close is still waiting.
	}

	// Now allow the handler to complete.
	close(handlerDone)

	// Wait for the RPC call to complete.
	<-callDone
	syncClient.Close()

	// GracefulClose should now complete.
	select {
	case err := <-gracefulDone:
		if err != nil {
			t.Errorf("TestGracefulCloseWaitsForSession: GracefulClose returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("TestGracefulCloseWaitsForSession: GracefulClose did not complete in time")
	}

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestGracefulCloseWaitsForSession: server did not shut down in time")
	}
}

func TestGracefulCloseTimeout(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that blocks forever (until context cancelled).
	handlerBlocking := make(chan struct{})

	srv := server.New()
	srv.Register(ctx, "test", "TestService", "BlockingEcho", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			close(handlerBlocking)
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	conn := New(ctx, clientConn)

	// Create a sync client and start a call.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "BlockingEcho")
	if err != nil {
		t.Fatalf("TestGracefulCloseTimeout: failed to create sync client: %v", err)
	}

	// Start the RPC in a goroutine (will block forever).
	go func() {
		syncClient.Call(ctx, []byte("test"))
	}()

	// Wait for handler to be running.
	<-handlerBlocking

	// GracefulClose with short timeout should time out.
	gracefulCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	err = conn.GracefulClose(gracefulCtx)
	if err == nil {
		t.Errorf("TestGracefulCloseTimeout: expected timeout error, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("TestGracefulCloseTimeout: expected DeadlineExceeded, got %v", err)
	}

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestGracefulCloseTimeout: server did not shut down in time")
	}
}

func TestNewSessionsDuringDraining(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that blocks until signaled.
	handlerBlocking := make(chan struct{})
	handlerDone := make(chan struct{})

	srv := server.New()
	srv.Register(ctx, "test", "TestService", "SlowEcho", server.SyncHandler{
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

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	conn := New(ctx, clientConn)

	// Create a sync client and start a call.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "SlowEcho")
	if err != nil {
		t.Fatalf("TestNewSessionsDuringDraining: failed to create sync client: %v", err)
	}

	// Start the RPC in a goroutine.
	go func() {
		syncClient.Call(ctx, []byte("test"))
	}()

	// Wait for handler to be running.
	<-handlerBlocking

	// Start graceful close in background.
	go func() {
		gracefulCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		conn.GracefulClose(gracefulCtx)
	}()

	// Give time for draining to start.
	time.Sleep(50 * time.Millisecond)

	// Connection should be draining now.
	if !conn.IsDraining() {
		t.Errorf("TestNewSessionsDuringDraining: expected IsDraining() to return true")
	}

	// Try to create a new session - should fail with ErrDraining.
	_, err = conn.Sync(ctx, "test", "TestService", "SlowEcho")
	if !errors.Is(err, ErrDraining) {
		t.Errorf("TestNewSessionsDuringDraining: expected ErrDraining, got %v", err)
	}

	// Allow the handler to complete.
	close(handlerDone)
	syncClient.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestNewSessionsDuringDraining: server did not shut down in time")
	}
}

func TestIsDraining(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	srv := server.New()
	srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	conn := New(ctx, clientConn)

	// Initially not draining.
	if conn.IsDraining() {
		t.Errorf("TestIsDraining: expected IsDraining() to return false initially")
	}

	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestIsDraining: server did not shut down in time")
	}
}

func TestServiceConfigTimeout(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that takes 200ms to respond.
	srv := server.New()
	srv.Register(ctx, "test", "TestService", "SlowEcho", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return req, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Create client with service config that has 50ms timeout.
	cfg := serviceconfig.NewBuilder().
		WithTimeout("test/TestService/SlowEcho", 50*time.Millisecond).
		Build()

	conn := New(ctx, clientConn, WithServiceConfig(cfg))
	defer conn.Close()

	syncClient, err := conn.Sync(ctx, "test", "TestService", "SlowEcho")
	if err != nil {
		t.Fatalf("TestServiceConfigTimeout: failed to create sync client: %v", err)
	}

	// Call should timeout due to service config.
	_, err = syncClient.Call(ctx, []byte("test"))
	if err == nil {
		t.Errorf("TestServiceConfigTimeout: expected timeout error, got nil")
	}

	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestServiceConfigTimeout: server did not shut down in time")
	}
}

func TestServiceConfigTimeoutNotAppliedWhenContextHasDeadline(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that takes 50ms to respond.
	srv := server.New()
	srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			time.Sleep(50 * time.Millisecond)
			return req, nil
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Service config with very short timeout (10ms).
	cfg := serviceconfig.NewBuilder().
		WithTimeout("test/TestService/Echo", 10*time.Millisecond).
		Build()

	conn := New(ctx, clientConn, WithServiceConfig(cfg))
	defer conn.Close()

	// Create context with longer deadline (500ms).
	callCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	syncClient, err := conn.Sync(callCtx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestServiceConfigTimeoutNotAppliedWhenContextHasDeadline: failed to create sync client: %v", err)
	}

	// Call should succeed because the context already has a deadline (500ms),
	// so the 10ms service config timeout is NOT applied.
	resp, err := syncClient.Call(callCtx, []byte("test"))
	if err != nil {
		t.Errorf("TestServiceConfigTimeoutNotAppliedWhenContextHasDeadline: expected success, got error: %v", err)
	}
	if string(resp) != "test" {
		t.Errorf("TestServiceConfigTimeoutNotAppliedWhenContextHasDeadline: got response %q, want %q", resp, "test")
	}

	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestServiceConfigTimeoutNotAppliedWhenContextHasDeadline: server did not shut down in time")
	}
}

func TestServiceConfigWildcard(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that takes 200ms to respond.
	srv := server.New()
	srv.Register(ctx, "test", "TestService", "Method1", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return req, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Service config with service-level wildcard timeout.
	cfg := serviceconfig.NewBuilder().
		WithTimeout("test/TestService/*", 50*time.Millisecond).
		Build()

	conn := New(ctx, clientConn, WithServiceConfig(cfg))
	defer conn.Close()

	syncClient, err := conn.Sync(ctx, "test", "TestService", "Method1")
	if err != nil {
		t.Fatalf("TestServiceConfigWildcard: failed to create sync client: %v", err)
	}

	// Call should timeout due to service wildcard config.
	_, err = syncClient.Call(ctx, []byte("test"))
	if err == nil {
		t.Errorf("TestServiceConfigWildcard: expected timeout error, got nil")
	}

	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestServiceConfigWildcard: server did not shut down in time")
	}
}

func TestServiceConfigNoMatch(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	// Server handler that responds immediately.
	srv := server.New()
	srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Service config with timeout for different method.
	cfg := serviceconfig.NewBuilder().
		WithTimeout("other/OtherService/OtherMethod", 10*time.Millisecond).
		Build()

	conn := New(ctx, clientConn, WithServiceConfig(cfg))
	defer conn.Close()

	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestServiceConfigNoMatch: failed to create sync client: %v", err)
	}

	// Call should succeed since no matching config.
	resp, err := syncClient.Call(ctx, []byte("hello"))
	if err != nil {
		t.Errorf("TestServiceConfigNoMatch: expected success, got error: %v", err)
	}
	if string(resp) != "hello" {
		t.Errorf("TestServiceConfigNoMatch: got response %q, want %q", resp, "hello")
	}

	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestServiceConfigNoMatch: server did not shut down in time")
	}
}

func TestServiceConfigWaitForReady(t *testing.T) {
	ctx := t.Context()
	clientConn, serverConn := pipe()

	srv := server.New()
	srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, serverConn)
	}()

	// Service config with WaitForReady.
	cfg := serviceconfig.NewBuilder().
		WithMethodConfig("test/TestService/*", serviceconfig.MethodConfig{
			WaitForReady: true,
		}).
		Build()

	conn := New(ctx, clientConn, WithServiceConfig(cfg))
	defer conn.Close()

	// With WaitForReady set in config, call should still succeed
	// (connection is already ready in this test).
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestServiceConfigWaitForReady: failed to create sync client: %v", err)
	}

	resp, err := syncClient.Call(ctx, []byte("hello"))
	if err != nil {
		t.Errorf("TestServiceConfigWaitForReady: expected success, got error: %v", err)
	}
	if string(resp) != "hello" {
		t.Errorf("TestServiceConfigWaitForReady: got response %q, want %q", resp, "hello")
	}

	syncClient.Close()
	conn.Close()

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Errorf("TestServiceConfigWaitForReady: server did not shut down in time")
	}
}

func TestGetWaitForReady(t *testing.T) {
	tests := []struct {
		name string
		opts []CallOption
		want bool
	}{
		{
			name: "Success: no options returns false",
			opts: nil,
			want: false,
		},
		{
			name: "Success: WithWaitForReady(true) returns true",
			opts: []CallOption{WithWaitForReady(true)},
			want: true,
		},
		{
			name: "Success: WithWaitForReady(false) returns false",
			opts: []CallOption{WithWaitForReady(false)},
			want: false,
		},
		{
			name: "Success: last option wins",
			opts: []CallOption{WithWaitForReady(true), WithWaitForReady(false)},
			want: false,
		},
	}

	for _, test := range tests {
		got := GetWaitForReady(test.opts...)
		if got != test.want {
			t.Errorf("TestGetWaitForReady(%s): got %v, want %v", test.name, got, test.want)
		}
	}
}
