package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/cert"
)

func TestIntegration_HttpProxy(t *testing.T) {
	// 1. Upstream Server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "true")
		w.WriteHeader(200)
		w.Write([]byte("hello upstream"))
	}))
	defer upstream.Close()

	// 2. Setup Proxy
	opts := &Options{
		Addr:              "127.0.0.1:9085", // Use explicit Loopback and different port
		StreamLargeBodies: 1024 * 1024,
	}
	p, err := NewProxy(opts)
	if err != nil {
		t.Fatalf("NewProxy error: %v", err)
	}
	
    // Start proxy
	go func() {
		if err := p.Start(); err != nil {
             t.Logf("Proxy start error: %v", err)
		}
	}()
    defer p.Close()
    
    // Wait for start
    time.Sleep(1 * time.Second)

	// 3. Client
    proxyUrl, _ := url.Parse("http://" + opts.Addr)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		},
	}

	// 4. Request
	resp, err := client.Get(upstream.URL)
	if err != nil {
		t.Fatalf("Client get error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello upstream" {
		t.Errorf("Want 'hello upstream', got '%s'", string(body))
	}
    if resp.Header.Get("X-Upstream") != "true" {
        t.Error("Missing X-Upstream header")
    }
}

func TestIntegration_GracefulShutdown(t *testing.T) {
	opts := &Options{
		Addr: ":0",
	}
	p, err := NewProxy(opts)
	if err != nil {
		t.Fatalf("NewProxy error: %v", err)
	}

	go func() {
		if err := p.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Proxy start error: %v", err)
		}
	}()
    
    // Wait for start
    time.Sleep(100 * time.Millisecond)
    
    // Shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    if err := p.Shutdown(ctx); err != nil {
        t.Errorf("Shutdown error: %v", err)
    }
}

func TestIntegration_HttpsConnect(t *testing.T) {
    // 1. Upstream HTTPS Server
    upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello https"))
	}))
	defer upstream.Close()
    
    // 2. Setup Proxy
    opts := &Options{
		Addr:              ":9081",
		StreamLargeBodies: 1024 * 1024,
        SslInsecure:       true, // Skip upstream verify
	}
    
    // Use a temp cert dir
    tmpDir := t.TempDir()
    ca, err := cert.NewSelfSignCA(tmpDir)
    if err != nil {
        t.Fatal(err)
    }
    opts.NewCaFunc = func() (cert.CA, error) {
        return ca, nil
    }
    
	p, err := NewProxy(opts)
	if err != nil {
		t.Fatalf("NewProxy error: %v", err)
	}
    
    go p.Start()
    defer p.Close()
    time.Sleep(500 * time.Millisecond)
    
    proxyUrl, _ := url.Parse("http://127.0.0.1:9081")
    
    // 3. Client that trusts the proxy CA (or skips verify)
    // To verify MITM, we need the client to trust the proxy's CA.
    // Since we generated a self-signed CA, we can add it to the client's pool.
    rootPEM := ca.GetRootCA()
    pool := x509.NewCertPool()
    // GetRootCA likely returns *x509.Certificate based on error.
    // If it returns *Certificate, we can't use AppendCertsFromPEM (takes []byte).
    // We should use AddCert.
    pool.AddCert(rootPEM)
    
    client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
            TLSClientConfig: &tls.Config{
                RootCAs: pool,
                InsecureSkipVerify: true,
            },
		},
	}
    
    // 4. Request
    resp, err := client.Get(upstream.URL)
    if err != nil {
        t.Fatalf("Client get https error: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        t.Errorf("Want 200, got %d", resp.StatusCode)
    }
    body, _ := io.ReadAll(resp.Body)
    if string(body) != "hello https" {
        t.Errorf("Want 'hello https', got '%s'", string(body))
    }
}
