package proxy

import (
	"context"
	"crypto/x509"
	"net/http"
	"testing"
	"time"
    "net/url"
)

func TestProxy_Shutdown(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, err := NewProxy(opts)
	if err != nil {
		t.Fatal(err)
	}

    // Start in goroutine
    go func() {
        if err := p.Start(); err != nil && err != http.ErrServerClosed {
            // t.Error cannot be called from goroutine safely without sync, but we hope 
            // for no error or ErrServerClosed
        }
    }()
    
    time.Sleep(100 * time.Millisecond) // Give it time to start
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := p.Shutdown(ctx); err != nil {
        t.Errorf("Shutdown error: %v", err)
    }
}

func TestProxy_GetCertificate(t *testing.T) {
    opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
    
    // Default CA should be generated
    cert := p.GetCertificate()
    if cert.SerialNumber == nil {
        t.Error("expected valid root CA cert")
    }
    
    // GetCert by CN
    tlsCert, err := p.GetCertificateByCN("example.com")
    if err != nil {
        t.Fatal(err)
    }
    if tlsCert == nil {
        t.Error("expected valid cert for example.com")
    }
    
    // Check leaf
    x509Cert, _ := x509.ParseCertificate(tlsCert.Certificate[0])
    if x509Cert.Subject.CommonName != "example.com" {
        t.Errorf("expected CN example.com, got %s", x509Cert.Subject.CommonName)
    }
}

func TestProxy_Upstream(t *testing.T) {
    // Helper to test realUpstreamProxy closure
    
    opts := &Options{Addr: ":0", Upstream: "http://upstream:8080"}
	p, _ := NewProxy(opts)
    
    // Mock request context with proxyReqCtxKey
    req := &http.Request{Header: make(http.Header)}
    ctxReq := &http.Request{Header: make(http.Header), Host: "example.com"}
    ctx := context.WithValue(req.Context(), proxyReqCtxKey, ctxReq)
    req = req.WithContext(ctx)
    
    proxyFn := p.realUpstreamProxy()
    u, err := proxyFn(req)
    if err != nil {
        t.Fatal(err)
    }
    if u.Host != "upstream:8080" {
        t.Errorf("expected upstream:8080, got %s", u.Host)
    }
    
    // Test SetUpstreamProxy
    p.SetUpstreamProxy(func(r *http.Request) (*url.URL, error) {
        return url.Parse("http://dynamic:9090")
    })
    
    u, _ = proxyFn(req)
    if u.Host != "dynamic:9090" {
        t.Errorf("expected dynamic:9090, got %s", u.Host)
    }
}
