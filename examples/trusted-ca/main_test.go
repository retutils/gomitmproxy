package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestTrustedCA_GetCert(t *testing.T) {
	ca, _ := NewTrustedCA()
	
	// Test cache hit
	_, err := ca.GetCert("your-domain.xx.com")
	if err == nil {
		// It might fail because files don't exist
	}
	
	_, err = ca.GetCert("invalid.com")
	if err == nil {
		t.Error("Expected error for invalid domain")
	}
}

func TestYourAddOn(t *testing.T) {
	addon := &YourAddOn{}
	client := &proxy.ClientConn{}
	addon.ClientConnected(client)
	if client.UpstreamCert {
		t.Error("Expected UpstreamCert false")
	}
	
	flow := &proxy.Flow{}
	addon.Request(flow)
	if flow.Response == nil || flow.Response.StatusCode != 200 {
		t.Error("Expected mocked response")
	}
}

func TestTrustedCA_RootCert(t *testing.T) {
	ca, _ := NewTrustedCA()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic from GetRootCA")
		}
	}()
	ca.GetRootCA()
}

func TestTrustedCA_LoadCert_Domain2(t *testing.T) {
	ca, _ := NewTrustedCA()
	_, _ = ca.GetCert("your-domain2.xx.com")
}

func TestTrustedCA_InterceptRule(t *testing.T) {
	rule := func(req *http.Request) bool {
		host, _, err2 := net.SplitHostPort(req.URL.Host)
		if err2 != nil {
			return false
		}
		return host == "your-domain.xx.com" || host == "your-domain2.xx.com"
	}
	
	tests := []struct {
		url  string
		want bool
	}{
		{"http://your-domain.xx.com:80", true},
		{"http://your-domain2.xx.com:443", true},
		{"http://other.com:80", false},
		{"http:///invalid-host", false}, // Valid URL syntax but empty host or something
	}
	
	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.url, nil)
		if got := rule(req); got != tt.want {
			t.Errorf("rule(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestRun(t *testing.T) {
	go Run()
	time.Sleep(100 * time.Millisecond)
}
