package main

import (
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestCloseConn(t *testing.T) {
	addon := &CloseConn{}
	client := &proxy.ClientConn{UpstreamCert: true}
	addon.ClientConnected(client)
	if client.UpstreamCert {
		t.Error("Expected UpstreamCert false")
	}

	f := &proxy.Flow{}
	addon.Requestheaders(f)
	if f.Response == nil || f.Response.StatusCode != 502 {
		t.Errorf("Expected 502 response, got %v", f.Response)
	}
}
