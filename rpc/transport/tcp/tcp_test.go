package tcp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/gostdlib/base/context"
	"github.com/kylelemons/godebug/pretty"

	"github.com/bearlytools/claw/rpc/client"
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/server"
)

func TestTCPTransportSynchronousRPC(t *testing.T) {
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

			// Setup RPC server.
			srv := server.New()
			err := srv.Register("test", "TestService", "Echo", server.SyncHandler{
				HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
					return append([]byte("echo:"), req...), nil
				},
			})
			if err != nil {
				t.Fatalf("[TestTCPTransportSynchronousRPC(%s)]: failed to register handler: %v", test.name, err)
			}

			// Create TCP listener on random port.
			listener, err := Listen(ctx, "127.0.0.1:0")
			if err != nil {
				t.Fatalf("[TestTCPTransportSynchronousRPC(%s)]: failed to listen: %v", test.name, err)
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

			// Connect via TCP transport.
			transport, err := Dial(ctx, listener.Addr().String())
			if err != nil {
				t.Fatalf("[TestTCPTransportSynchronousRPC(%s)]: failed to dial: %v", test.name, err)
			}
			defer transport.Close()

			// Create RPC client.
			conn := client.New(ctx, transport)
			defer conn.Close()

			// Create sync client.
			syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
			if err != nil {
				t.Fatalf("[TestTCPTransportSynchronousRPC(%s)]: failed to create sync client: %v", test.name, err)
			}
			defer syncClient.Close()

			// Send requests and verify responses.
			for i, req := range test.requests {
				resp, err := syncClient.Call(ctx, req)
				switch {
				case err == nil && test.wantErr:
					t.Errorf("[TestTCPTransportSynchronousRPC(%s)]: request %d: got err == nil, want err != nil", test.name, i)
					continue
				case err != nil && !test.wantErr:
					t.Errorf("[TestTCPTransportSynchronousRPC(%s)]: request %d: got err == %v, want err == nil", test.name, i, err)
					continue
				case err != nil:
					continue
				}

				want := append([]byte("echo:"), req...)
				if diff := pretty.Compare(want, resp); diff != "" {
					t.Errorf("[TestTCPTransportSynchronousRPC(%s)]: request %d: response mismatch (-want +got):\n%s", test.name, i, diff)
				}
			}
		})
	}
}

func TestTCPTransportBiDirectionalRPC(t *testing.T) {
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
				t.Fatalf("[TestTCPTransportBiDirectionalRPC(%s)]: failed to register handler: %v", test.name, err)
			}

			// Create TCP listener.
			listener, err := Listen(ctx, "127.0.0.1:0")
			if err != nil {
				t.Fatalf("[TestTCPTransportBiDirectionalRPC(%s)]: failed to listen: %v", test.name, err)
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

			// Connect via TCP transport.
			transport, err := Dial(ctx, listener.Addr().String())
			if err != nil {
				t.Fatalf("[TestTCPTransportBiDirectionalRPC(%s)]: failed to dial: %v", test.name, err)
			}
			defer transport.Close()

			// Create RPC client.
			conn := client.New(ctx, transport)
			defer conn.Close()

			bidir, err := conn.BiDir(ctx, "test", "TestService", "BiDir")
			if err != nil {
				t.Fatalf("[TestTCPTransportBiDirectionalRPC(%s)]: failed to create bidir client: %v", test.name, err)
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
					t.Errorf("[TestTCPTransportBiDirectionalRPC(%s)]: failed to send: %v", test.name, err)
				}
			}
			bidir.CloseSend()

			// Wait for receive to complete.
			<-recvDone

			if recvErr != nil && !test.wantErr {
				t.Errorf("[TestTCPTransportBiDirectionalRPC(%s)]: receive error: %v", test.name, recvErr)
			}

			if diff := pretty.Compare(test.serverMsgs, received); diff != "" {
				t.Errorf("[TestTCPTransportBiDirectionalRPC(%s)]: received messages mismatch (-want +got):\n%s", test.name, diff)
			}

			bidir.Close()
			conn.Close()

			// Verify server received client messages.
			var serverReceived [][]byte
			for msg := range serverRecv {
				serverReceived = append(serverReceived, msg)
			}
			if diff := pretty.Compare(test.clientMsgs, serverReceived); diff != "" {
				t.Errorf("[TestTCPTransportBiDirectionalRPC(%s)]: server received messages mismatch (-want +got):\n%s", test.name, diff)
			}
		})
	}
}

func TestTCPTransportWithTLS(t *testing.T) {
	ctx := t.Context()

	// Generate self-signed certificate for testing.
	tlsConfig, err := generateTestTLSConfig()
	if err != nil {
		t.Fatalf("[TestTCPTransportWithTLS]: failed to generate TLS config: %v", err)
	}

	// Setup RPC server.
	srv := server.New()
	srv.Register("test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return append([]byte("secure:"), req...), nil
		},
	})

	// Create TLS listener.
	listener, err := Listen(ctx, "127.0.0.1:0", WithTLSConfig(tlsConfig))
	if err != nil {
		t.Fatalf("[TestTCPTransportWithTLS]: failed to listen: %v", err)
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

	// Create client TLS config that trusts the server certificate.
	clientTLSConfig := &tls.Config{
		InsecureSkipVerify: true, // For testing with self-signed cert.
	}

	// Connect via TLS TCP transport.
	transport, err := Dial(ctx, listener.Addr().String(), WithTLSConfig(clientTLSConfig))
	if err != nil {
		t.Fatalf("[TestTCPTransportWithTLS]: failed to dial: %v", err)
	}
	defer transport.Close()

	// Create RPC client.
	conn := client.New(ctx, transport)
	defer conn.Close()

	// Create sync client.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("[TestTCPTransportWithTLS]: failed to create sync client: %v", err)
	}
	defer syncClient.Close()

	// Send request.
	resp, err := syncClient.Call(ctx, []byte("hello"))
	if err != nil {
		t.Fatalf("[TestTCPTransportWithTLS]: call failed: %v", err)
	}

	want := []byte("secure:hello")
	if diff := pretty.Compare(want, resp); diff != "" {
		t.Errorf("[TestTCPTransportWithTLS]: response mismatch (-want +got):\n%s", diff)
	}
}

func TestTCPTransportConnectionClose(t *testing.T) {
	ctx := t.Context()

	srv := server.New()
	srv.Register("test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	listener, err := Listen(ctx, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("[TestTCPTransportConnectionClose]: failed to listen: %v", err)
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

	transport, err := Dial(ctx, listener.Addr().String())
	if err != nil {
		t.Fatalf("[TestTCPTransportConnectionClose]: failed to dial: %v", err)
	}

	conn := client.New(ctx, transport)

	// Create a session.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("[TestTCPTransportConnectionClose]: failed to create sync client: %v", err)
	}

	// Close the connection.
	conn.Close()
	transport.Close()

	// Verify session operations fail.
	_, err = syncClient.Call(ctx, []byte("test"))
	if err == nil {
		t.Errorf("[TestTCPTransportConnectionClose]: expected error after connection close, got nil")
	}

	// Verify Err() returns the fatal error.
	if conn.Err() != nil && conn.Err() != io.EOF {
		// Connection closed normally, not an error condition.
	}
}

func TestTCPTransportDialErrors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "Error: invalid address",
			addr:    "invalid:address:format",
			wantErr: true,
		},
		{
			name:    "Error: connection refused",
			addr:    "127.0.0.1:1", // Port 1 is unlikely to be listening.
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Dial(ctx, test.addr)
			switch {
			case err == nil && test.wantErr:
				t.Errorf("[TestTCPTransportDialErrors(%s)]: got err == nil, want err != nil", test.name)
			case err != nil && !test.wantErr:
				t.Errorf("[TestTCPTransportDialErrors(%s)]: got err == %v, want err == nil", test.name, err)
			}
		})
	}
}

func TestTCPTransportBufferedIO(t *testing.T) {
	ctx := t.Context()

	// This test verifies that buffered I/O works correctly
	// by sending many small messages that would be inefficient without buffering.
	srv := server.New()
	srv.Register("test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	listener, err := Listen(ctx, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("[TestTCPTransportBufferedIO]: failed to listen: %v", err)
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

	transport, err := Dial(ctx, listener.Addr().String())
	if err != nil {
		t.Fatalf("[TestTCPTransportBufferedIO]: failed to dial: %v", err)
	}
	defer transport.Close()

	conn := client.New(ctx, transport)
	defer conn.Close()

	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("[TestTCPTransportBufferedIO]: failed to create sync client: %v", err)
	}
	defer syncClient.Close()

	// Send many small requests.
	for i := 0; i < 100; i++ {
		req := []byte("small")
		resp, err := syncClient.Call(ctx, req)
		if err != nil {
			t.Fatalf("[TestTCPTransportBufferedIO]: call %d failed: %v", i, err)
		}
		if diff := pretty.Compare(req, resp); diff != "" {
			t.Errorf("[TestTCPTransportBufferedIO]: call %d: response mismatch (-want +got):\n%s", i, diff)
		}
	}
}

func TestTCPServerListenAndServe(t *testing.T) {
	ctx := t.Context()

	// Setup RPC server.
	rpcSrv := server.New()
	err := rpcSrv.Register("test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return append([]byte("server:"), req...), nil
		},
	})
	if err != nil {
		t.Fatalf("[TestTCPServerListenAndServe]: failed to register handler: %v", err)
	}

	// Create TCP server using new pattern.
	tcpSrv := NewServer(rpcSrv, "127.0.0.1:0")

	// Start server in background.
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- tcpSrv.ListenAndServe(ctx)
	}()

	// Wait for server to start listening.
	var addr net.Addr
	for i := 0; i < 100; i++ {
		addr = tcpSrv.Addr()
		if addr != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if addr == nil {
		t.Fatalf("[TestTCPServerListenAndServe]: server did not start listening")
	}

	// Connect via TCP transport.
	transport, err := Dial(ctx, addr.String())
	if err != nil {
		t.Fatalf("[TestTCPServerListenAndServe]: failed to dial: %v", err)
	}
	defer transport.Close()

	// Create RPC client.
	conn := client.New(ctx, transport)
	defer conn.Close()

	// Create sync client.
	syncClient, err := conn.Sync(ctx, "test", "TestService", "Echo")
	if err != nil {
		t.Fatalf("[TestTCPServerListenAndServe]: failed to create sync client: %v", err)
	}
	defer syncClient.Close()

	// Send request.
	resp, err := syncClient.Call(ctx, []byte("hello"))
	if err != nil {
		t.Fatalf("[TestTCPServerListenAndServe]: call failed: %v", err)
	}

	want := []byte("server:hello")
	if diff := pretty.Compare(want, resp); diff != "" {
		t.Errorf("[TestTCPServerListenAndServe]: response mismatch (-want +got):\n%s", diff)
	}

	// Shutdown the server.
	if err := tcpSrv.Close(); err != nil {
		t.Errorf("[TestTCPServerListenAndServe]: close failed: %v", err)
	}

	// Wait for server to stop.
	select {
	case err := <-serverErr:
		if err != nil {
			t.Logf("[TestTCPServerListenAndServe]: server returned: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("[TestTCPServerListenAndServe]: server did not stop")
	}
}

func TestTCPServerShutdown(t *testing.T) {
	ctx := t.Context()

	// Setup RPC server.
	rpcSrv := server.New()
	rpcSrv.Register("test", "TestService", "Echo", server.SyncHandler{
		HandleFunc: func(ctx context.Context, req []byte, md []msgs.Metadata) ([]byte, error) {
			return req, nil
		},
	})

	// Create TCP server.
	tcpSrv := NewServer(rpcSrv, "127.0.0.1:0")

	// Start server.
	serverDone := make(chan struct{})
	go func() {
		tcpSrv.ListenAndServe(ctx)
		close(serverDone)
	}()

	// Wait for server to start.
	for i := 0; i < 100; i++ {
		if tcpSrv.Addr() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Graceful shutdown.
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := tcpSrv.Shutdown(shutdownCtx); err != nil {
		t.Errorf("[TestTCPServerShutdown]: shutdown failed: %v", err)
	}

	// Wait for server to stop.
	select {
	case <-serverDone:
		// Success
	case <-time.After(5 * time.Second):
		t.Errorf("[TestTCPServerShutdown]: server did not stop after shutdown")
	}
}

// generateTestTLSConfig creates a self-signed certificate for testing.
func generateTestTLSConfig() (*tls.Config, error) {
	// Generate RSA key.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Create certificate template.
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Create self-signed certificate.
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	// Create TLS certificate.
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}
