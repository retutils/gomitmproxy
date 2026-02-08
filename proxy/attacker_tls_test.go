package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/cert"
)

func TestAttacker_HttpsTlsDial_Integration(t *testing.T) {
	opts := &Options{Addr: ":0", SslInsecure: true}
	p, _ := NewProxy(opts)
	a, _ := newAttacker(p)
	ca, _ := cert.NewSelfSignCA("")
	a.ca = ca

	go func() {
		a.start()
	}()

	clientConn, proxyClientConn := net.Pipe()
	proxyUpstreamConn, upstreamConn := net.Pipe()

	// 3. Setup Context
	wc := newWrapClientConn(proxyClientConn, p)
	// We need to setup ServerConn because serverTlsHandshake relies on it
	ws := &wrapServerConn{
		Conn: proxyUpstreamConn,
		proxy: p,
	}
	serverConn := newServerConn()
	serverConn.Conn = ws
	
	connCtx := &ConnContext{
		ClientConn: &ClientConn{
			Conn: wc,
			Tls:  true,
		},
		ServerConn: serverConn,
		proxy: p,
	}
	wc.connCtx = connCtx
	ws.connCtx = connCtx
	
	ctx := context.WithValue(context.Background(), connContextKey, connCtx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		a.httpsTlsDial(ctx, wc, proxyUpstreamConn)
	}()

	clientTlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "example.com",
		NextProtos:         []string{"http/1.1"},
	}
	clientTls := tls.Client(clientConn, clientTlsConfig)

	go func() {
		clientTls.Handshake()
	}()

	// Mock Upstream Server Handshake
	upstreamTlsConfig := &tls.Config{
		Certificates: []tls.Certificate{testCert(t)},
		NextProtos:   []string{"http/1.1"},
	}
	serverTls := tls.Server(upstreamConn, upstreamTlsConfig)
	if err := serverTls.Handshake(); err != nil {
		t.Errorf("Upstream handshake failed: %v", err)
	}

	// We expect client handshake to complete eventually
	// But since this is a pipe, we need to read/write concurrently?
	// net.Pipe is synchronous. Writing blocks until reading.
	// httpsTlsDial handles proxyClientConn and proxyUpstreamConn concurrently.
	// It proxies clientHello to server (upstream), waits for serverHello, returns to client.
	
	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	clientTls.Close()
	serverTls.Close()
	<-done
}

func testCert(t *testing.T) tls.Certificate {
	ca, err := cert.NewSelfSignCA("")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ca.GetCert("example.com")
	if err != nil {
		t.Fatal(err)
	}
	return *c
}

func TestAttacker_HttpsLazyAttack_Integration(t *testing.T) {
	opts := &Options{Addr: ":0", SslInsecure: true}
	p, _ := NewProxy(opts)
	a, _ := newAttacker(p)
	defer a.server.Close()
	ca, _ := cert.NewSelfSignCA("")
	a.ca = ca
	go func() {
		a.start()
	}()

	// Use TCP instead of Pipe to avoid potential deadlocks with TLS handshake
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Connect client in goroutine
	clientErr := make(chan error, 1)
	go func() {
		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			clientErr <- err
			return
		}
		defer conn.Close()

		clientTlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "example.com",
			NextProtos:         []string{"http/1.1"},
		}
		clientTls := tls.Client(conn, clientTlsConfig)
		if err := clientTls.Handshake(); err != nil {
			// Expected to potentially fail or succeed
		}
		clientErr <- nil
	}()

	proxyClientConn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer proxyClientConn.Close()
	
	wc := newWrapClientConn(proxyClientConn, p)
	connCtx := &ConnContext{
		ClientConn: &ClientConn{
			Conn: wc,
			Tls:  true,
		},
		proxy: p,
	}
	wc.connCtx = connCtx
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		a.httpsLazyAttack(ctx, wc, req)
	}()

	// Wait for attack logic to theoretically finish (it won't return until connection closed)
	// We check for client completion
	select {
	case <-clientErr:
	case <-time.After(2 * time.Second):
		t.Fatal("Client timeout")
	}
	
	// Close proxy side to force return
	proxyClientConn.Close()
	
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		// It might take moment
	}
}

func TestAttacker_HttpsTlsDial_ClientHandshakeError(t *testing.T) {
	opts := &Options{Addr: ":0", SslInsecure: true}
	p, _ := NewProxy(opts)
	a, _ := newAttacker(p)
	ca, _ := cert.NewSelfSignCA("")
	a.ca = ca

	clientConn, proxyClientConn := net.Pipe()
	proxyUpstreamConn, _ := net.Pipe()

	wc := newWrapClientConn(proxyClientConn, p)
	
	// Create minimal ConnCtx
	connCtx := &ConnContext{
		ClientConn: &ClientConn{
			Conn: wc,
			Tls:  true,
		},
		proxy: p,
	}
	wc.connCtx = connCtx
	// ServerConn not strictly needed for client handshake error path unless it uses it before error?
	// httpsTlsDial uses a.serverTlsHandshake which uses ServerConn.
	// But client handshake happens concurrently.
	// If client handshake fails, errChan1 receives error.
	// Then it closes connections and returns.
	
	ctx := context.WithValue(context.Background(), connContextKey, connCtx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// We expect this to return error via logging and closing connection
		a.httpsTlsDial(ctx, wc, proxyUpstreamConn)
	}()

	// Client sends garbage to fail handshake
	go func() {
		clientConn.Write([]byte("garbage"))
		clientConn.Close()
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout")
	}
}

func TestAttacker_HttpsTlsDial_ServerHandshakeError(t *testing.T) {
	opts := &Options{Addr: ":0", SslInsecure: true}
	p, _ := NewProxy(opts)
	a, _ := newAttacker(p)
	ca, _ := cert.NewSelfSignCA("")
	a.ca = ca
	
	go func() { a.start() }()
	defer a.server.Close()

	clientConn, proxyClientConn := net.Pipe()
	proxyUpstreamConn, upstreamConn := net.Pipe()

	wc := newWrapClientConn(proxyClientConn, p)
	
	// Setup ServerConn pointing to upstream
	ws := &wrapServerConn{
		Conn: proxyUpstreamConn,
		proxy: p,
	}
	serverConn := newServerConn()
	serverConn.Conn = ws
	
	connCtx := &ConnContext{
		ClientConn: &ClientConn{
			Conn: wc,
			Tls:  true,
		},
		ServerConn: serverConn,
		proxy: p,
	}
	wc.connCtx = connCtx
	ws.connCtx = connCtx
	ctx := context.WithValue(context.Background(), connContextKey, connCtx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		a.httpsTlsDial(ctx, wc, proxyUpstreamConn)
	}()

	// Client handshake attempts to succeed
	clientTlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "example.com",
		NextProtos:         []string{"http/1.1"},
	}
	clientTls := tls.Client(clientConn, clientTlsConfig)
	
	go func() {
		clientTls.Handshake()
		clientTls.Close()
	}()

	// Upstream connection fails handshake (garbage or close)
	go func() {
		upstreamConn.Close()
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout")
	}
}
