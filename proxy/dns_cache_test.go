package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProxy_DNSCache_Integration(t *testing.T) {
	opts := &Options{
		Addr: ":0",
	}
	p, err := NewProxy(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	req := &http.Request{
		Method: "GET",
		URL:    u,
		Host:   u.Host,
	}

	// First call - should resolve and connect
	conn1, err := p.getUpstreamConn(context.Background(), req)
	if err != nil {
		t.Fatalf("First dial failed: %v", err)
	}
	conn1.Close()

	// Second call - should use cache (internal to fastdialer) and connect
	conn2, err := p.getUpstreamConn(context.Background(), req)
	if err != nil {
		t.Fatalf("Second dial failed: %v", err)
	}
	conn2.Close()
}

func TestProxy_DNSRetry_Fail(t *testing.T) {
	opts := &Options{
		Addr:       ":0",
		DnsRetries: 2,
	}
	p, _ := NewProxy(opts)

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "http", Host: "non-existent.invalid"},
		Host:   "non-existent.invalid",
	}

	_, err := p.getUpstreamConn(context.Background(), req)
	if err == nil {
		t.Error("Expected error for non-existent host")
	}
}
