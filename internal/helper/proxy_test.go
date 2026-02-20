package helper

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestGetProxyConn_HTTP(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			w.WriteHeader(http.StatusOK)
			hj, ok := w.(http.Hijacker)
			if !ok {
				return
			}
			conn, _, _ := hj.Hijack()
			defer conn.Close()
			conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
			io.Copy(conn, conn)
		}
	}))
	defer proxyServer.Close()

	proxyUrl, _ := url.Parse(proxyServer.URL)
	ctx := context.Background()
	conn, err := GetProxyConn(ctx, proxyUrl, "example.com:80", false)
	if err == nil {
		conn.Close()
	}
}

func TestGetProxyConn_HTTPS(t *testing.T) {
	proxyServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer proxyServer.Close()

	proxyUrl, _ := url.Parse(proxyServer.URL)
	ctx := context.Background()
	conn, err := GetProxyConn(ctx, proxyUrl, "example.com:80", true)
	if err == nil {
		conn.Close()
	}
}

func TestGetProxyConn_HTTP_Auth(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Proxy-Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusProxyAuthRequired)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()

	u, _ := url.Parse(proxyServer.URL)
	u.User = url.UserPassword("user", "pass")
	
	conn, err := GetProxyConn(context.Background(), u, "example.com:80", false)
	if err == nil {
		conn.Close()
	}
}

func TestGetProxyConn_Timeout(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer proxyServer.Close()

	u, _ := url.Parse(proxyServer.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	_, err := GetProxyConn(ctx, u, "example.com:80", false)
	if err == nil {
		t.Error("Expected timeout error")
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

func TestGetProxyConn_HTTPS_FailHandshake(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer proxyServer.Close()

	u, _ := url.Parse(proxyServer.URL)
	u.Scheme = "https"
	
	_, err := GetProxyConn(context.Background(), u, "example.com:80", false)
	if err == nil {
		t.Error("Expected error for non-TLS server acting as HTTPS proxy")
	}
}

func TestGetProxyConn_HTTP_InvalidResponse(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		defer conn.Close()
		conn.Write([]byte("not an http response"))
	}))
	defer proxyServer.Close()

	u, _ := url.Parse(proxyServer.URL)
	_, err := GetProxyConn(context.Background(), u, "example.com:80", false)
	if err == nil {
		t.Error("Expected error for invalid HTTP response from proxy")
	}
}

func TestGetProxyConn_Socks5_Error(t *testing.T) {
	u, _ := url.Parse("socks5://user:pass@localhost:1")
	_, err := GetProxyConn(context.Background(), u, "example.com:80", false)
	if err == nil {
		t.Error("Expected error for invalid socks5")
	}
}

func TestGetProxyConn_Error(t *testing.T) {
	ctx := context.Background()
	proxyUrl, _ := url.Parse("http://localhost:1")
	_, err := GetProxyConn(ctx, proxyUrl, "example.com:80", false)
	if err == nil {
		t.Error("Expected error for invalid proxy address")
	}
}
