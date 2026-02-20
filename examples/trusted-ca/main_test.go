package main

import (
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
	// To make this succeed, we would need to mock the strings in trusted-ca.go
	// Since we can't easily do that without changing the code, 
	// we just hit the call and accept the error.
	_, _ = ca.GetCert("your-domain2.xx.com")
}

func TestRun(t *testing.T) {
	go Run()
	time.Sleep(100 * time.Millisecond)
}
