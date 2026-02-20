package proxy

import (
	"bytes"
	"net"
	"net/http"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/atomic"
)

func TestBaseAddon_Coverage(t *testing.T) {
	addon := &BaseAddon{}
	client := &ClientConn{}
	connCtx := &ConnContext{}
	flow := &Flow{}
	
	// Just call all methods to ensure coverage
	addon.ClientConnected(client)
	addon.ClientDisconnected(client)
	addon.ServerConnected(connCtx)
	addon.ServerDisconnected(connCtx)
	addon.TlsEstablishedServer(connCtx)
	addon.Requestheaders(flow)
	addon.Request(flow)
	addon.Responseheaders(flow)
	addon.Response(flow)
	
	reader := bytes.NewReader([]byte("test"))
	if addon.StreamRequestModifier(flow, reader) != reader {
		t.Error("StreamRequestModifier should return input")
	}
	if addon.StreamResponseModifier(flow, reader) != reader {
		t.Error("StreamResponseModifier should return input")
	}
	
	addon.AccessProxyServer(nil, nil)
	addon.WebsocketHandshake(flow)
	addon.WebsocketMessage(flow, nil)
}

func TestLogAddon_Coverage(t *testing.T) {
	addon := &LogAddon{}
	
	// Mock objects with minimal requirements
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
	mockConn := &addonDummyConn{remoteAddr: addr, localAddr: addr}
	
	client := &ClientConn{Conn: mockConn}
	server := &ServerConn{Conn: mockConn, Address: "example.com"}
	connCtx := &ConnContext{
		ClientConn: client,
		ServerConn: server,
		FlowCount:  *atomic.NewUint32(5),
	}
	
	reqObj, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	flow := &Flow{
		Id:          uuid.NewV4(),
		ConnContext: connCtx,
		Request:     NewRequest(reqObj),
		done:        make(chan struct{}),
	}
	
	addon.ClientConnected(client)
	addon.ClientDisconnected(client)
	addon.ServerConnected(connCtx)
	addon.ServerDisconnected(connCtx)
	addon.Requestheaders(flow)
	
	// Simulate flow finish for the goroutine in Requestheaders
	flow.Response = &Response{StatusCode: 200, Body: []byte("ok")}
	flow.Finish()
	
	// Give the goroutine time to run
	time.Sleep(50 * time.Millisecond)
	
	addon.WebsocketHandshake(flow)
	addon.WebsocketMessage(flow, &WebSocketMessage{FromClient: true, Type: 1, Data: []byte("hello")})
}

func TestUpstreamCertAddon_Coverage(t *testing.T) {
	addon := NewUpstreamCertAddon(true)
	conn := &ClientConn{}
	addon.ClientConnected(conn)
	if !conn.UpstreamCert {
		t.Error("Expected UpstreamCert true")
	}
}

type addonDummyConn struct {
	net.Conn
	remoteAddr net.Addr
	localAddr  net.Addr
}

func (d *addonDummyConn) RemoteAddr() net.Addr { return d.remoteAddr }
func (d *addonDummyConn) LocalAddr() net.Addr  { return d.localAddr }
func (d *addonDummyConn) Close() error         { return nil }
func (d *addonDummyConn) Write(b []byte) (int, error) { return len(b), nil }
