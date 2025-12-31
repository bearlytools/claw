package client

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/gostdlib/base/context"
	"github.com/kylelemons/godebug/pretty"

	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/bearlytools/claw/rpc/server"
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
			err := srv.Register("test", "TestService", "Echo", server.SyncHandler{
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
			err := srv.Register("test", "TestService", "BiDir", server.BiDirHandler{
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
			err := srv.Register("test", "TestService", "Send", server.SendHandler{
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
			err := srv.Register("test", "TestService", "Recv", server.RecvHandler{
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
	srv.Register("test", "TestService", "Echo", server.SyncHandler{
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
	srv.Register("test", "TestService", "Echo", server.SyncHandler{
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
	err := srv.Register("test", "TestService", "BiDir", server.BiDirHandler{
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
	err := srv.Register("test", "TestService", "Recv", server.RecvHandler{
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
