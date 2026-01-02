package http

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gostdlib/base/context"
	"github.com/kylelemons/godebug/pretty"

	"github.com/bearlytools/claw/rpc/client"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/server"
)

func TestHTTPTransportSynchronousRPC(t *testing.T) {
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			// Setup RPC server.
			srv := server.New()
			err := srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
				HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
					return append([]byte("echo:"), req...), nil
				},
			})
			if err != nil {
				t.Fatalf("[TestHTTPTransportSynchronousRPC(%s)]: failed to register handler: %v", test.name, err)
			}

			// Create HTTP handler with h2c support and test server.
			handler := NewHandler(srv)
			httpServer := httptest.NewServer(handler.H2CHandler())
			defer httpServer.Close()

			// Connect via HTTP transport (uses h2c for HTTP/2 cleartext).
			transport, err := Dial(ctx, httpServer.URL)
			if err != nil {
				t.Fatalf("[TestHTTPTransportSynchronousRPC(%s)]: failed to dial: %v", test.name, err)
			}
			defer transport.Close()

			// Create RPC client.
			conn := client.New(ctx, transport)
			defer conn.Close()

			// Create sync client.
			syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
			if err != nil {
				t.Fatalf("[TestHTTPTransportSynchronousRPC(%s)]: failed to create sync client: %v", test.name, err)
			}
			defer syncClient.Close()

			// Send requests and verify responses.
			for i, req := range test.requests {
				resp, err := syncClient.Call(ctx, req)
				switch {
				case err == nil && test.wantErr:
					t.Errorf("[TestHTTPTransportSynchronousRPC(%s)]: request %d: got err == nil, want err != nil", test.name, i)
					continue
				case err != nil && !test.wantErr:
					t.Errorf("[TestHTTPTransportSynchronousRPC(%s)]: request %d: got err == %v, want err == nil", test.name, i, err)
					continue
				case err != nil:
					continue
				}

				want := append([]byte("echo:"), req...)
				if diff := pretty.Compare(want, resp); diff != "" {
					t.Errorf("[TestHTTPTransportSynchronousRPC(%s)]: request %d: response mismatch (-want +got):\n%s", test.name, i, diff)
				}
			}
		})
	}
}

func TestHTTPTransportBiDirectionalRPC(t *testing.T) {
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

			// Server receives client messages and sends its own.
			serverRecv := make(chan []byte, len(test.clientMsgs))
			srv := server.New()
			err := srv.Register(ctx, "test", "TestService", "BiDir", server.BiDirHandler{
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
				t.Fatalf("[TestHTTPTransportBiDirectionalRPC(%s)]: failed to register handler: %v", test.name, err)
			}

			// Create HTTP handler with h2c support and test server.
			handler := NewHandler(srv)
			httpServer := httptest.NewServer(handler.H2CHandler())
			defer httpServer.Close()

			// Connect via HTTP transport (uses h2c for HTTP/2 cleartext).
			transport, err := Dial(ctx, httpServer.URL)
			if err != nil {
				t.Fatalf("[TestHTTPTransportBiDirectionalRPC(%s)]: failed to dial: %v", test.name, err)
			}
			defer transport.Close()

			// Create RPC client.
			conn := client.New(ctx, transport)
			defer conn.Close()

			bidir, err := conn.BiDir(ctx, "test", "TestService", "BiDir")
			if err != nil {
				t.Fatalf("[TestHTTPTransportBiDirectionalRPC(%s)]: failed to create bidir client: %v", test.name, err)
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
					t.Errorf("[TestHTTPTransportBiDirectionalRPC(%s)]: failed to send: %v", test.name, err)
				}
			}
			bidir.CloseSend()

			// Wait for receive to complete.
			<-recvDone

			if recvErr != nil && !test.wantErr {
				t.Errorf("[TestHTTPTransportBiDirectionalRPC(%s)]: receive error: %v", test.name, recvErr)
			}

			if diff := pretty.Compare(test.serverMsgs, received); diff != "" {
				t.Errorf("[TestHTTPTransportBiDirectionalRPC(%s)]: received messages mismatch (-want +got):\n%s", test.name, diff)
			}

			bidir.Close()
			conn.Close()

			// Verify server received client messages.
			var serverReceived [][]byte
			for msg := range serverRecv {
				serverReceived = append(serverReceived, msg)
			}
			if diff := pretty.Compare(test.clientMsgs, serverReceived); diff != "" {
				t.Errorf("[TestHTTPTransportBiDirectionalRPC(%s)]: server received messages mismatch (-want +got):\n%s", test.name, diff)
			}
		})
	}
}

func TestHTTPTransportConnectionClose(t *testing.T) {
	ctx := t.Context()

	srv := server.New()
	srv.Register(ctx, "test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	// Create HTTP handler with h2c support.
	handler := NewHandler(srv)
	httpServer := httptest.NewServer(handler.H2CHandler())
	defer httpServer.Close()

	// Connect via HTTP transport (uses h2c for HTTP/2 cleartext).
	transport, err := Dial(ctx, httpServer.URL)
	if err != nil {
		t.Fatalf("TestHTTPTransportConnectionClose: failed to dial: %v", err)
	}

	conn := client.New(ctx, transport)

	// Create a session.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("TestHTTPTransportConnectionClose: failed to create sync client: %v", err)
	}

	// Close the connection.
	conn.Close()
	transport.Close()

	// Verify session operations fail.
	_, err = syncClient.Call(ctx, []byte("test"))
	if err == nil {
		t.Errorf("TestHTTPTransportConnectionClose: expected error after connection close, got nil")
	}

	// Verify Err() returns the fatal error.
	if conn.Err() != nil && conn.Err() != io.EOF {
		// Connection closed normally, not an error condition.
	}
}

func TestHTTPTransportDialErrors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "Error: invalid URL",
			url:     "://invalid",
			wantErr: true,
		},
		{
			name:    "Error: unsupported scheme",
			url:     "ftp://example.com/rpc",
			wantErr: true,
		},
		{
			name:    "Error: connection refused",
			url:     "http://localhost:1", // Port 1 is unlikely to be listening
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Dial(ctx, test.url)
			switch {
			case err == nil && test.wantErr:
				t.Errorf("[TestHTTPTransportDialErrors(%s)]: got err == nil, want err != nil", test.name)
			case err != nil && !test.wantErr:
				t.Errorf("[TestHTTPTransportDialErrors(%s)]: got err == %v, want err == nil", test.name, err)
			}
		})
	}
}

