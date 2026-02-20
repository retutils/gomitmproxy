package helper

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetProxyConn_HTTP(t *testing.T) {
	// Mock HTTP Proxy Server that handles CONNECT
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			w.WriteHeader(http.StatusOK)
			hj, ok := w.(http.Hijacker)
			if !ok {
				return
			}
			conn, _, _ := hj.Hijack()
			defer conn.Close()
			
			// After 200 OK, the proxy acts as a tunnel.
			// Just write something to indicate it's open
			conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
			
			// Echo everything back
			io.Copy(conn, conn)
		}
	}))
	defer proxyServer.Close()

	proxyUrl, _ := url.Parse(proxyServer.URL)
	ctx := context.Background()
	
	// Test normal CONNECT
	conn, err := GetProxyConn(ctx, proxyUrl, "example.com:80", false)
	if err == nil {
		conn.Close()
	}
}

func TestGetProxyConn_Socks5(t *testing.T) {
    // SOCKS5 is harder to mock without a server. 
    // We can at least test the error path for invalid address.
    ctx := context.Background()
    proxyUrl, _ := url.Parse("socks5://user:pass@localhost:1")
    _, err := GetProxyConn(ctx, proxyUrl, "example.com:80", false)
    if err == nil {
        t.Error("Expected error for invalid socks5 proxy")
    }
}

func TestGetProxyConn_HTTPS(t *testing.T) {
	// Mock HTTPS Proxy Server
	proxyServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer proxyServer.Close()

	proxyUrl, _ := url.Parse(proxyServer.URL)
	ctx := context.Background()
	
	// Use TLSServer's URL which is https
	conn, err := GetProxyConn(ctx, proxyUrl, "example.com:80", true)
	if err == nil {
		conn.Close()
	}
}

func TestGetProxyConn_HTTP_Fail(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer proxyServer.Close()

	proxyUrl, _ := url.Parse(proxyServer.URL)
	ctx := context.Background()
	
	_, err := GetProxyConn(ctx, proxyUrl, "example.com:80", false)
	if err == nil {
		t.Error("Expected error for 403 response")
	}
}
