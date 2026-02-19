package proxy

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
    "fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func handleError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatal(err)
    }
}

type testProxyHelper struct {
    server         *http.Server
    proxyAddr      string
    ln             net.Listener
    tlsPlainLn     net.Listener
    tlsLn          net.Listener
    httpEndpoint   string
    httpsEndpoint  string
    testProxy      *Proxy
    getProxyClient func() *http.Client
}

func (h *testProxyHelper) init(t *testing.T) {
    t.Helper()

    // Setup backend server listener
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    handleError(t, err)
    h.ln = ln
    h.httpEndpoint = fmt.Sprintf("http://%s", ln.Addr().String())

    // Setup TLS listener
    // We need a cert for ServeTLS? 
    // Usually ServeTLS takes certFile/keyFile.
    // Tests use empty strings? `go helper.server.ServeTLS(helper.tlsPlainLn, "", "")`.
    // This implies GenerateSelfSignedCert inside or ignores it?
    // StartTLS doc: "If certFile and keyFile are not provided, the server's certificate is used."
    // Server has TLSConfig.
    // Tests set `helper.server.TLSConfig`.
    
    tlsPlainLn, err := net.Listen("tcp", "127.0.0.1:0")
    handleError(t, err)
    h.tlsPlainLn = tlsPlainLn
    h.httpsEndpoint = fmt.Sprintf("https://%s", tlsPlainLn.Addr().String())
    
    tlsLn, err := net.Listen("tcp", "127.0.0.1:0")
    handleError(t, err)
    h.tlsLn = tlsLn
    
    // Setup server handler if not set
    if h.server.Handler == nil {
        h.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Type", "text/plain")
            w.WriteHeader(200)
            w.Write([]byte("OK"))
        })
    }
    
    if h.server.TLSConfig == nil {
        h.server.TLSConfig = &tls.Config{}
    }
    
    // Generate self-signed cert
    if len(h.server.TLSConfig.Certificates) == 0 {
        // Generate key
        priv, err := rsa.GenerateKey(rand.Reader, 2048)
        handleError(t, err)

        template := x509.Certificate{
            SerialNumber: big.NewInt(1),
            Subject: pkix.Name{
                Organization: []string{"Test Co"},
            },
            NotBefore: time.Now(),
            NotAfter:  time.Now().Add(time.Hour),

            KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
            ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
            BasicConstraintsValid: true,
        }
        
        // Hosts
        template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))
        
        derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
        handleError(t, err)

        cert := tls.Certificate{
            Certificate: [][]byte{derBytes},
            PrivateKey:  priv,
        }
        h.server.TLSConfig.Certificates = append(h.server.TLSConfig.Certificates, cert)
    }
    
    // Setup Proxy
    opts := &Options{
        Addr:              h.proxyAddr,
        StreamLargeBodies: 1024 * 1024,
        SslInsecure:       true,
    }
    p, err := NewProxy(opts)
    handleError(t, err)
    h.testProxy = p
    
    h.getProxyClient = func() *http.Client {
        proxyUrl, _ := url.Parse("http://127.0.0.1" + h.proxyAddr)
        return &http.Client{
            Transport: &http.Transport{
                Proxy: http.ProxyURL(proxyUrl),
                TLSClientConfig: &tls.Config{
                    InsecureSkipVerify: true,
                },
            },
        }
    }
}

// Ensure dummy cert generation for server if needed
// For now, assume TestConnection manages server TLS logic implicitly or uses a specific setup I might be missing.
// But wait, TestConnection calls `ServeTLS` with empty strings. 
// Go's `ServeTLS` documentation says: "If the certificate and key are not provided, ServeTLS uses the server's TLSConfig.Certificates".
// So I should populate TLSConfig with a generated cert if empty.
// I'll skip that for now and see if it runs (hoping TestConnection does valid setup or I get a cert generation error I can fix).

// My new tests

func TestProxySetters(t *testing.T) {
	opts := &Options{
		Addr: ":0",
	}
	p, err := NewProxy(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Test SetShouldInterceptRule
	rule := func(req *http.Request) bool { return true }
	p.SetShouldInterceptRule(rule)

	// Test SetUpstreamProxy
    u, _ := url.Parse("http://proxy.example.com:8888")
	proxyFn := func(req *http.Request) (*url.URL, error) {
        return u, nil
    }
	p.SetUpstreamProxy(proxyFn)
	
    // Verify
    req := &http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}}
	gotU, err := p.getUpstreamProxyUrl(req)
	if err != nil {
		t.Error(err)
	}
	if gotU.String() != u.String() {
		t.Errorf("want %s, got %s", u.String(), gotU.String())
	}

	// Test SetAuthProxy
	auth := func(res http.ResponseWriter, req *http.Request) (bool, error) { return true, nil }
	p.SetAuthProxy(auth)
}

func TestProxy_GetUpstreamConn_Error(t *testing.T) {
    opts := &Options{Addr: ":0"}
    p, _ := NewProxy(opts)
    
    // Invalid host
    req := &http.Request{URL: &url.URL{Scheme: "http", Host: "invalid-host.local"}}
    _, err := p.getUpstreamConn(context.Background(), req)
    if err == nil {
        t.Error("Expected error for invalid host")
    }

    // Proxy error
    p.Opts.Upstream = "http://localhost:1" // Invalid proxy
    _, err = p.getUpstreamConn(context.Background(), req)
    if err == nil {
        t.Error("Expected error for invalid proxy")
    }
}

func TestProxyUpstream(t *testing.T) {
    opts := &Options{Addr: ":0"}
    p, _ := NewProxy(opts)
    
    // Test realUpstreamProxy
    _ = p.realUpstreamProxy()
}

func TestProxy_StartError(t *testing.T) {
    // Port 1 is usually restricted or unavailable, but better to use something that definitely fails
    // or just close it.
    opts := &Options{Addr: "127.0.0.1:1"}
    p, _ := NewProxy(opts)
    err := p.Start()
    if err == nil {
        t.Error("Expected error when starting on port 1")
    }
}

func TestProxy_Addr(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	if p.Addr() != ":0" {
		t.Errorf("Expected :0 Addr before Start, got %s", p.Addr())
	}
	go p.Start()
	defer p.Close()
	time.Sleep(100 * time.Millisecond)
	if p.Addr() == ":0" || p.Addr() == "" {
		t.Errorf("Expected dynamic Addr after Start, got %s", p.Addr())
	}
}

func TestProxy_Certificate(t *testing.T) {
    opts := &Options{Addr: ":0"}
    p, _ := NewProxy(opts)
    
    cert := p.GetCertificate()
    if cert.Subject.CommonName != "mitmproxy" {
        t.Errorf("Unexpected root cert CN: %s", cert.Subject.CommonName)
    }
    
    c, err := p.GetCertificateByCN("example.com")
    if err != nil {
        t.Error(err)
    }
    if c == nil {
        t.Error("Expected cert for example.com")
    }
}
