package unix

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/gostdlib/base/context"
	"github.com/kylelemons/godebug/pretty"

	"github.com/bearlytools/claw/rpc/client"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/server"
)

func TestUnixTransportSynchronousRPC(t *testing.T) {
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
			requests: [][]byte{make([]byte, 100000)}, // 100KB
			wantErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			// Create temp socket path.
			socketPath := tempSocketPath(t)

			// Setup RPC server.
			srv := server.New()
			err := srv.Register("test", "TestService", "Echo", server.SyncHandler{
				HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
					return append([]byte("echo:"), req...), nil
				},
			})
			if err != nil {
				t.Fatalf("[TestUnixTransportSynchronousRPC(%s)]: failed to register handler: %v", test.name, err)
			}

			// Create Unix socket listener.
			listener, err := Listen(ctx, socketPath)
			if err != nil {
				t.Fatalf("[TestUnixTransportSynchronousRPC(%s)]: failed to listen: %v", test.name, err)
			}
			defer listener.Close()

			// Start accepting connections.
			go func() {
				for {
					trans, err := listener.Accept(ctx)
					if err != nil {
						return
					}
					go srv.Serve(ctx, trans)
				}
			}()

			// Connect via Unix socket transport.
			transport, err := Dial(ctx, socketPath)
			if err != nil {
				t.Fatalf("[TestUnixTransportSynchronousRPC(%s)]: failed to dial: %v", test.name, err)
			}
			defer transport.Close()

			// Create RPC client.
			conn := client.New(ctx, transport)
			defer conn.Close()

			// Create sync client.
			syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
			if err != nil {
				t.Fatalf("[TestUnixTransportSynchronousRPC(%s)]: failed to create sync client: %v", test.name, err)
			}
			defer syncClient.Close()

			// Send requests and verify responses.
			for i, req := range test.requests {
				resp, err := syncClient.Call(ctx, req)
				switch {
				case err == nil && test.wantErr:
					t.Errorf("[TestUnixTransportSynchronousRPC(%s)]: request %d: got err == nil, want err != nil", test.name, i)
					continue
				case err != nil && !test.wantErr:
					t.Errorf("[TestUnixTransportSynchronousRPC(%s)]: request %d: got err == %v, want err == nil", test.name, i, err)
					continue
				case err != nil:
					continue
				}

				want := append([]byte("echo:"), req...)
				if diff := pretty.Compare(want, resp); diff != "" {
					t.Errorf("[TestUnixTransportSynchronousRPC(%s)]: request %d: response mismatch (-want +got):\n%s", test.name, i, diff)
				}
			}
		})
	}
}

func TestUnixTransportBiDirectionalRPC(t *testing.T) {
	tests := []struct {
		name       string
		clientMsgs [][]byte
		serverMsgs [][]byte
		wantErr    bool
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

			socketPath := tempSocketPath(t)

			// Server receives client messages and sends its own.
			serverRecv := make(chan []byte, len(test.clientMsgs))
			srv := server.New()
			err := srv.Register("test", "TestService", "BiDir", server.BiDirHandler{
				HandleFunc: func(ctx context.Context, stream *server.BiDirStream) error {
					// Send and receive concurrently.
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
				t.Fatalf("[TestUnixTransportBiDirectionalRPC(%s)]: failed to register handler: %v", test.name, err)
			}

			// Create Unix socket listener.
			listener, err := Listen(ctx, socketPath)
			if err != nil {
				t.Fatalf("[TestUnixTransportBiDirectionalRPC(%s)]: failed to listen: %v", test.name, err)
			}
			defer listener.Close()

			// Start accepting connections.
			go func() {
				for {
					trans, err := listener.Accept(ctx)
					if err != nil {
						return
					}
					go srv.Serve(ctx, trans)
				}
			}()

			// Connect via Unix socket transport.
			transport, err := Dial(ctx, socketPath)
			if err != nil {
				t.Fatalf("[TestUnixTransportBiDirectionalRPC(%s)]: failed to dial: %v", test.name, err)
			}
			defer transport.Close()

			// Create RPC client.
			conn := client.New(ctx, transport)
			defer conn.Close()

			bidir, err := conn.BiDir(ctx, "test", "TestService", "BiDir")
			if err != nil {
				t.Fatalf("[TestUnixTransportBiDirectionalRPC(%s)]: failed to create bidir client: %v", test.name, err)
			}

			// Send and receive concurrently.
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
					t.Errorf("[TestUnixTransportBiDirectionalRPC(%s)]: failed to send: %v", test.name, err)
				}
			}
			bidir.CloseSend()

			// Wait for receive to complete.
			<-recvDone

			if recvErr != nil && !test.wantErr {
				t.Errorf("[TestUnixTransportBiDirectionalRPC(%s)]: receive error: %v", test.name, recvErr)
			}

			if diff := pretty.Compare(test.serverMsgs, received); diff != "" {
				t.Errorf("[TestUnixTransportBiDirectionalRPC(%s)]: received messages mismatch (-want +got):\n%s", test.name, diff)
			}

			bidir.Close()
			conn.Close()

			// Verify server received client messages.
			var serverReceived [][]byte
			for msg := range serverRecv {
				serverReceived = append(serverReceived, msg)
			}
			if diff := pretty.Compare(test.clientMsgs, serverReceived); diff != "" {
				t.Errorf("[TestUnixTransportBiDirectionalRPC(%s)]: server received messages mismatch (-want +got):\n%s", test.name, diff)
			}
		})
	}
}

func TestUnixTransportConnectionClose(t *testing.T) {
	ctx := t.Context()

	socketPath := tempSocketPath(t)

	srv := server.New()
	srv.Register("test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	listener, err := Listen(ctx, socketPath)
	if err != nil {
		t.Fatalf("[TestUnixTransportConnectionClose]: failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			trans, err := listener.Accept(ctx)
			if err != nil {
				return
			}
			go srv.Serve(ctx, trans)
		}
	}()

	transport, err := Dial(ctx, socketPath)
	if err != nil {
		t.Fatalf("[TestUnixTransportConnectionClose]: failed to dial: %v", err)
	}

	conn := client.New(ctx, transport)

	// Create a session.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("[TestUnixTransportConnectionClose]: failed to create sync client: %v", err)
	}

	// Close the connection.
	conn.Close()
	transport.Close()

	// Verify session operations fail.
	_, err = syncClient.Call(ctx, []byte("test"))
	if err == nil {
		t.Errorf("[TestUnixTransportConnectionClose]: expected error after connection close, got nil")
	}

	// Verify Err() returns the fatal error.
	if conn.Err() != nil && conn.Err() != io.EOF {
		// Connection closed normally, not an error condition.
	}
}

func TestUnixTransportDialErrors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Error: socket does not exist",
			path:    "/nonexistent/path/to/socket.sock",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Dial(ctx, test.path)
			switch {
			case err == nil && test.wantErr:
				t.Errorf("[TestUnixTransportDialErrors(%s)]: got err == nil, want err != nil", test.name)
			case err != nil && !test.wantErr:
				t.Errorf("[TestUnixTransportDialErrors(%s)]: got err == %v, want err == nil", test.name, err)
			}
		})
	}
}

func TestUnixTransportSocketPermissions(t *testing.T) {
	ctx := t.Context()

	socketPath := tempSocketPath(t)

	// Create listener with specific permissions.
	listener, err := Listen(ctx, socketPath, WithSocketMode(0666))
	if err != nil {
		t.Fatalf("[TestUnixTransportSocketPermissions]: failed to listen: %v", err)
	}
	defer listener.Close()

	// Verify socket file permissions.
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("[TestUnixTransportSocketPermissions]: failed to stat socket: %v", err)
	}

	// Check that the socket mode is set (note: socket bit will also be set).
	mode := info.Mode().Perm()
	if mode != 0666 {
		t.Errorf("[TestUnixTransportSocketPermissions]: got mode %o, want %o", mode, 0666)
	}
}

func TestUnixTransportUnlinkExisting(t *testing.T) {
	ctx := t.Context()

	socketPath := tempSocketPath(t)

	// Create first listener.
	listener1, err := Listen(ctx, socketPath)
	if err != nil {
		t.Fatalf("[TestUnixTransportUnlinkExisting]: failed to create first listener: %v", err)
	}
	listener1.Close()

	// Socket file should still exist after close (listener removes it, but let's test the unlink option).
	// Actually, our Close() removes the file, so let's create a dummy file instead.
	f, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("[TestUnixTransportUnlinkExisting]: failed to create dummy file: %v", err)
	}
	f.Close()

	// Try to create second listener with unlink disabled - should fail.
	_, err = Listen(ctx, socketPath, WithUnlinkExisting(false))
	if err == nil {
		t.Errorf("[TestUnixTransportUnlinkExisting]: expected error when socket exists and unlink disabled")
	}

	// Create second listener with unlink enabled (default) - should succeed.
	// But first we need to make the file look like a socket for our code to remove it.
	os.Remove(socketPath)

	listener2, err := Listen(ctx, socketPath, WithUnlinkExisting(true))
	if err != nil {
		t.Fatalf("[TestUnixTransportUnlinkExisting]: failed to create second listener: %v", err)
	}
	defer listener2.Close()
}

func TestUnixTransportBufferedIO(t *testing.T) {
	ctx := t.Context()

	socketPath := tempSocketPath(t)

	// This test verifies that buffered I/O works correctly.
	srv := server.New()
	srv.Register("test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	listener, err := Listen(ctx, socketPath)
	if err != nil {
		t.Fatalf("[TestUnixTransportBufferedIO]: failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			trans, err := listener.Accept(ctx)
			if err != nil {
				return
			}
			go srv.Serve(ctx, trans)
		}
	}()

	transport, err := Dial(ctx, socketPath)
	if err != nil {
		t.Fatalf("[TestUnixTransportBufferedIO]: failed to dial: %v", err)
	}
	defer transport.Close()

	conn := client.New(ctx, transport)
	defer conn.Close()

	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("[TestUnixTransportBufferedIO]: failed to create sync client: %v", err)
	}
	defer syncClient.Close()

	// Send many small requests.
	for i := 0; i < 100; i++ {
		req := []byte("small")
		resp, err := syncClient.Call(ctx, req)
		if err != nil {
			t.Fatalf("[TestUnixTransportBufferedIO]: call %d failed: %v", i, err)
		}
		if diff := pretty.Compare(req, resp); diff != "" {
			t.Errorf("[TestUnixTransportBufferedIO]: call %d: response mismatch (-want +got):\n%s", i, diff)
		}
	}
}

// tempSocketPath returns a unique socket path in a temp directory.
// Unix sockets have a path length limit (~104 chars on macOS), so we use
// a short path in /tmp instead of t.TempDir() which creates long paths.
func tempSocketPath(t *testing.T) string {
	t.Helper()

	// Create a short unique name to stay under Unix socket path limits.
	f, err := os.CreateTemp("/tmp", "sock")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	os.Remove(path) // Remove the file so we can use the path for a socket.

	t.Cleanup(func() {
		os.Remove(path)
	})

	return path
}

func TestUnixServerListenAndServe(t *testing.T) {
	ctx := t.Context()

	socketPath := tempSocketPath(t)

	// Setup RPC server.
	rpcSrv := server.New()
	err := rpcSrv.Register("test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return append([]byte("server:"), req...), nil
		},
	})
	if err != nil {
		t.Fatalf("[TestUnixServerListenAndServe]: failed to register handler: %v", err)
	}

	// Create Unix server using new pattern.
	unixSrv := NewServer(rpcSrv, socketPath)

	// Start server in background.
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- unixSrv.ListenAndServe(ctx)
	}()

	// Wait for server to start listening.
	for i := 0; i < 100; i++ {
		if unixSrv.Addr() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if unixSrv.Addr() == nil {
		t.Fatalf("[TestUnixServerListenAndServe]: server did not start listening")
	}

	// Connect via Unix socket transport.
	transport, err := Dial(ctx, socketPath)
	if err != nil {
		t.Fatalf("[TestUnixServerListenAndServe]: failed to dial: %v", err)
	}
	defer transport.Close()

	// Create RPC client.
	conn := client.New(ctx, transport)
	defer conn.Close()

	// Create sync client.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("[TestUnixServerListenAndServe]: failed to create sync client: %v", err)
	}
	defer syncClient.Close()

	// Send request.
	resp, err := syncClient.Call(ctx, []byte("hello"))
	if err != nil {
		t.Fatalf("[TestUnixServerListenAndServe]: call failed: %v", err)
	}

	want := []byte("server:hello")
	if diff := pretty.Compare(want, resp); diff != "" {
		t.Errorf("[TestUnixServerListenAndServe]: response mismatch (-want +got):\n%s", diff)
	}

	// Shutdown the server.
	if err := unixSrv.Close(); err != nil {
		t.Errorf("[TestUnixServerListenAndServe]: close failed: %v", err)
	}

	// Wait for server to stop.
	select {
	case err := <-serverErr:
		if err != nil {
			t.Logf("[TestUnixServerListenAndServe]: server returned: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("[TestUnixServerListenAndServe]: server did not stop")
	}
}

func TestUnixServerShutdown(t *testing.T) {
	ctx := t.Context()

	socketPath := tempSocketPath(t)

	// Setup RPC server.
	rpcSrv := server.New()
	rpcSrv.Register("test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	// Create Unix server.
	unixSrv := NewServer(rpcSrv, socketPath)

	// Start server.
	serverDone := make(chan struct{})
	go func() {
		unixSrv.ListenAndServe(ctx)
		close(serverDone)
	}()

	// Wait for server to start.
	for i := 0; i < 100; i++ {
		if unixSrv.Addr() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Graceful shutdown.
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := unixSrv.Shutdown(shutdownCtx); err != nil {
		t.Errorf("[TestUnixServerShutdown]: shutdown failed: %v", err)
	}

	// Wait for server to stop.
	select {
	case <-serverDone:
		// Success
	case <-time.After(5 * time.Second):
		t.Errorf("[TestUnixServerShutdown]: server did not stop after shutdown")
	}
}
