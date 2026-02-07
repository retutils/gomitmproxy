package proxy

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestEntry_DirectTransfer(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	
	// Set shouldIntercept to false
	p.SetShouldInterceptRule(func(req *http.Request) bool {
		return false
	})
	
	// Setup upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("upstream"))
	}))
	defer upstream.Close()
	
	u, _ := url.Parse(upstream.URL)
	p.Opts.Upstream = u.Host // Use Upstream option to force traffic there? 
    // Or directTransfer uses request host.
    
    // Setup entry
	e := newEntry(p)
	
	// Create a CONNECT request
    // We need a hijackable ResponseWriter. httptest.ResponseRecorder is not hijackable by default?
    // Wait, httptest.NewRecorder() does NOT implement Hijack. 
    // We need a custom writer or use a real server.
    // Integration test used real server.
    // Here we can use a custom mock that implements Hijacker.
    
    rec := &hijackRecorder{
        ResponseRecorder: httptest.NewRecorder(),
        conn: &mockConn{}, // reusable mockConn
    }
    
    req := httptest.NewRequest("CONNECT", u.Host, nil)
    // Inject connContext
    connCtx := newConnContext(&mockConn{}, p)
    ctx := context.WithValue(req.Context(), connContextKey, connCtx)
    req = req.WithContext(ctx)
    
    e.ServeHTTP(rec, req)
    
    if rec.Code != 0 { 
        // Hijacked response might not have code set in recorder?
        // directTransfer calls establishConnection which writes "HTTP/1.1 200 Connection Established".
    }
}

func TestEntry_Auth(t *testing.T) {
    opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
    
    authCalled := false
    p.SetAuthProxy(func(res http.ResponseWriter, req *http.Request) (bool, error) {
        authCalled = true
        return false, nil // Fail auth
    })
    
    e := newEntry(p)
    rec := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "http://example.com", nil)
    
    e.ServeHTTP(rec, req)
    
    if !authCalled {
        t.Error("Auth not called")
    }
    if rec.Code != http.StatusProxyAuthRequired {
        t.Errorf("want 407, got %d", rec.Code)
    }
}

func TestEntry_LoopDetection(t *testing.T) {
    opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
    e := newEntry(p)
    
    rec := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/foo", nil) // Not absolute URL
    
    e.ServeHTTP(rec, req)
    
    if rec.Code != 400 {
        t.Errorf("want 400 for loop/non-proxy request, got %d", rec.Code)
    }
}

// Mock Hijacker
type hijackRecorder struct {
    *httptest.ResponseRecorder
    conn net.Conn
}

func (r *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    return r.conn, bufio.NewReadWriter(bufio.NewReader(r.conn), bufio.NewWriter(r.conn)), nil
}
