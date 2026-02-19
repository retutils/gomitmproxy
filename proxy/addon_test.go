package proxy

import (
	"bytes"
	"net/url"
	"testing"

	"go.uber.org/atomic"
)

func TestBaseAddon(t *testing.T) {
	addon := &BaseAddon{}
	// Call all methods to ensure no panic
	addon.ClientConnected(nil)
	addon.ClientDisconnected(nil)
	addon.ServerConnected(nil)
	addon.ServerDisconnected(nil)
	addon.TlsEstablishedServer(nil)
	addon.Requestheaders(nil)
	addon.Request(nil)
	addon.Responseheaders(nil)
	addon.Response(nil)
	if addon.StreamRequestModifier(nil, nil) != nil {
		// returns in
	}
	if addon.StreamResponseModifier(nil, nil) != nil {
		// returns in
	}
	addon.AccessProxyServer(nil, nil)
	addon.WebsocketHandshake(nil)
	addon.WebsocketMessage(nil, nil)
}

func TestLogAddon(t *testing.T) {
	addon := &LogAddon{}
	
	// Create minimal mock objects
	cconn := &ClientConn{
		Conn: &mockConn{},
	}
	sconn := &ServerConn{
		Conn: &mockConn{},
	}
	connCtx := &ConnContext{
		ClientConn: cconn,
		ServerConn: sconn,
        FlowCount: *atomic.NewUint32(0),
	}
	
	// ClientConnected
	addon.ClientConnected(cconn)
	
	// ClientDisconnected
	addon.ClientDisconnected(cconn)
	
	// ServerConnected
	addon.ServerConnected(connCtx)
	
	// ServerDisconnected
	addon.ServerDisconnected(connCtx)
	
	// Requestheaders
	u, _ := url.Parse("http://example.com")
	f := NewFlow()
	f.Request = &Request{Method: "GET", URL: u}
	f.ConnContext = connCtx
	
	addon.Requestheaders(f)
	
	// WebsocketHandshake
	addon.WebsocketHandshake(f)
	
	// WebsocketMessage
	addon.WebsocketMessage(f, &WebSocketMessage{Data: []byte("test"), Type: 1, FromClient: true})

	// Wait for async log in Requestheaders (it waits for f.Done())
	f.Response = &Response{StatusCode: 200, Body: []byte("OK")}
	f.Finish()

    // Response
    addon.Response(f)
}

func TestUpstreamCertAddon(t *testing.T) {
	addon := NewUpstreamCertAddon(true)
	if !addon.UpstreamCert {
		t.Error("Expected UpstreamCert true")
	}
	
	cconn := &ClientConn{}
	addon.ClientConnected(cconn)
	if !cconn.UpstreamCert {
		t.Error("Expected ClientConn.UpstreamCert to be set")
	}
}

func TestLogAddon_StreamModifier(t *testing.T) {
	addon := &LogAddon{}
	r := bytes.NewBufferString("test")
	if addon.StreamRequestModifier(nil, r) != r {
		t.Error("StreamRequestModifier should return input reader")
	}
	if addon.StreamResponseModifier(nil, r) != r {
		t.Error("StreamResponseModifier should return input reader")
	}
}
