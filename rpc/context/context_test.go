package context

import (
	"net"
	"testing"
)

func TestRemoteAddrNotSet(t *testing.T) {
	ctx := t.Context()

	addr := RemoteAddr(ctx)
	if addr != nil {
		t.Errorf("TestRemoteAddrNotSet: got %v, want nil", addr)
	}
}

func TestWithRemoteAddrTCP(t *testing.T) {
	ctx := t.Context()

	tcpAddr := &net.TCPAddr{
		IP:   net.ParseIP("192.168.1.100"),
		Port: 54321,
	}

	ctx = WithRemoteAddr(ctx, tcpAddr)

	got := RemoteAddr(ctx)
	if got == nil {
		t.Fatal("TestWithRemoteAddrTCP: got nil, want non-nil")
	}

	gotTCP, ok := got.(*net.TCPAddr)
	if !ok {
		t.Fatalf("TestWithRemoteAddrTCP: got type %T, want *net.TCPAddr", got)
	}

	if !gotTCP.IP.Equal(tcpAddr.IP) {
		t.Errorf("TestWithRemoteAddrTCP: got IP %v, want %v", gotTCP.IP, tcpAddr.IP)
	}

	if gotTCP.Port != tcpAddr.Port {
		t.Errorf("TestWithRemoteAddrTCP: got port %d, want %d", gotTCP.Port, tcpAddr.Port)
	}
}

func TestWithRemoteAddrUDP(t *testing.T) {
	ctx := t.Context()

	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP("10.0.0.50"),
		Port: 12345,
	}

	ctx = WithRemoteAddr(ctx, udpAddr)

	got := RemoteAddr(ctx)
	if got == nil {
		t.Fatal("TestWithRemoteAddrUDP: got nil, want non-nil")
	}

	gotUDP, ok := got.(*net.UDPAddr)
	if !ok {
		t.Fatalf("TestWithRemoteAddrUDP: got type %T, want *net.UDPAddr", got)
	}

	if !gotUDP.IP.Equal(udpAddr.IP) {
		t.Errorf("TestWithRemoteAddrUDP: got IP %v, want %v", gotUDP.IP, udpAddr.IP)
	}
}

func TestWithRemoteAddrIPv6(t *testing.T) {
	ctx := t.Context()

	tcpAddr := &net.TCPAddr{
		IP:   net.ParseIP("2001:db8::1"),
		Port: 8080,
	}

	ctx = WithRemoteAddr(ctx, tcpAddr)

	got := RemoteAddr(ctx)
	if got == nil {
		t.Fatal("TestWithRemoteAddrIPv6: got nil, want non-nil")
	}

	gotTCP, ok := got.(*net.TCPAddr)
	if !ok {
		t.Fatalf("TestWithRemoteAddrIPv6: got type %T, want *net.TCPAddr", got)
	}

	if !gotTCP.IP.Equal(tcpAddr.IP) {
		t.Errorf("TestWithRemoteAddrIPv6: got IP %v, want %v", gotTCP.IP, tcpAddr.IP)
	}
}

func TestRemoteAddrDoesNotAffectParent(t *testing.T) {
	parentCtx := t.Context()

	tcpAddr := &net.TCPAddr{
		IP:   net.ParseIP("192.168.1.100"),
		Port: 54321,
	}

	childCtx := WithRemoteAddr(parentCtx, tcpAddr)

	// Parent should not have the address
	if addr := RemoteAddr(parentCtx); addr != nil {
		t.Errorf("TestRemoteAddrDoesNotAffectParent: parent ctx got %v, want nil", addr)
	}

	// Child should have the address
	if addr := RemoteAddr(childCtx); addr == nil {
		t.Error("TestRemoteAddrDoesNotAffectParent: child ctx got nil, want non-nil")
	}
}
