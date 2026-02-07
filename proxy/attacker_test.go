package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
    "crypto/tls"

    uuid "github.com/satori/go.uuid"
    "github.com/sirupsen/logrus"
)

// Mock generic connection
type mockConn struct {}
func (m *mockConn) Read(b []byte) (n int, err error) { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (m *mockConn) Close() error { return nil }
func (m *mockConn) LocalAddr() net.Addr { return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0} }
func (m *mockConn) RemoteAddr() net.Addr { return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0} }
func (m *mockConn) SetDeadline(t time.Time) error { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// Mock connection that fails on Read
type failReadConn struct {
	mockConn
}

func (f *failReadConn) Read(b []byte) (n int, err error) {
	return 0, errors.New("read error")
}

// Mock connection that fails on Write
type failWriteConn struct {
	mockConn
}

func (f *failWriteConn) Write(b []byte) (n int, err error) {
	return 0, errors.New("write error")
}

func TestAttacker_All(t *testing.T) {
	opts := &Options{
		Addr: ":0",
	}
	p, err := NewProxy(opts)
	if err != nil {
		t.Fatal(err)
	}
	a := p.attacker
    p.Addons = nil

	// Test httpsDial with error
	// Mock a request with context
	req := httptest.NewRequest("CONNECT", "https://example.com:443", nil)
    
    // Inject ConnContext
    connCtx := &ConnContext{
        proxy: p,
        ClientConn: &ClientConn{
            Id: uuid.NewV4(),
            Conn: &mockConn{},
        },
    }
    ctx := context.WithValue(req.Context(), connContextKey, connCtx)
    req = req.WithContext(ctx)
	
	// Case 1: Dial error (unreachable)
	// We can't easily mock net.Dial inside attacker without changing attacker structure or using an interface.
	// But we can try to dial a closed port.
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond) // use our ctx with value
	defer cancel()
	// Use 127.0.0.1:0 (usually fails immediately or connects to nothing?)
	// Actually :0 binds to random free port.
	// Use a reserved IP or closed port.
	// localhost:1 is likely closed.
	req.URL.Host = "127.0.0.1:1"
	
	_, err = a.httpsDial(ctx, req)
	if err == nil {
		t.Error("expected dial error, got nil")
	}

    // Test start/close
    // a.start() is already tested in integration
    // a.Close()
    // a.Addr()
}

func TestAttacker_Reply_Errors(t *testing.T) {
    opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	a := p.attacker
    
    // Mock ResponseWriter that fails
    // httptest.ResponseRecorder doesn't fail.
    // We need a custom ResponseWriter.
    
    // Test reply with nil body
    rec := httptest.NewRecorder()
    resp := &Response{
        StatusCode: 200,
        Header: make(http.Header),
    }
    
    // Use logger
    logEntry := logrus.NewEntry(logrus.New())
    
    a.reply(rec, logEntry, resp, nil)
    if rec.Code != 200 {
        t.Errorf("want 200, got %d", rec.Code)
    }
    
    // Test reply with body reader error
    rec = httptest.NewRecorder()
    resp.BodyReader = &failReader{}
    a.reply(rec, logEntry, resp, nil)
    // Should log error but not panic
    
    // Test reply with body bytes write error (Response.Body)
    resp.BodyReader = nil
    resp.Body = []byte("short")
    failW := &failWriter{}
    // a.reply takes http.ResponseWriter. failWriter needs to implement it?
    // Header(), Write(), WriteHeader()
    
    a.reply(failW, logEntry, resp, nil)
}

type failReader struct{}
func (f *failReader) Read(p []byte) (n int, err error) { return 0, errors.New("reader fail") }

type failWriter struct {
    header http.Header
}
func (f *failWriter) Header() http.Header { 
    if f.header == nil { f.header = make(http.Header) }
    return f.header 
}
func (f *failWriter) Write([]byte) (int, error) { return 0, errors.New("writer fail") }
func (f *failWriter) WriteHeader(statusCode int) {}

func TestAttacker_Handshake_Errors(t *testing.T) {
    opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	a := p.attacker
    
    ctx := context.Background()
    
    // Test serverTlsHandshake with failed read conn
    fConn := &failReadConn{}
    connCtx := &ConnContext{
        ServerConn: &ServerConn{Conn: fConn},
        ClientConn: &ClientConn{
            clientHello: &tls.ClientHelloInfo{ServerName: "example.com"},
        },
    }
    err := a.serverTlsHandshake(ctx, connCtx)
    if err == nil {
        t.Error("expected handshake error on read fail")
    }
    
    // Test serverTlsHandshake with failed write conn
    wConn := &failWriteConn{}
    connCtx.ServerConn.Conn = wConn
    
    err = a.serverTlsHandshake(ctx, connCtx)
    if err == nil {
         t.Error("expected handshake error on write fail")
    }
}
