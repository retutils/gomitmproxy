package main

import (
	"net/url"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestRewriteHost(t *testing.T) {
	addon := &RewriteHost{}
	client := &proxy.ClientConn{UpstreamCert: true}
	addon.ClientConnected(client)
	if client.UpstreamCert {
		t.Error("Expected UpstreamCert false")
	}

	f := &proxy.Flow{
		Request: &proxy.Request{
			URL: &url.URL{Scheme: "https", Host: "example.com"},
		},
	}
	addon.Requestheaders(f)
	if f.Request.URL.Host != "www.baidu.com" || f.Request.URL.Scheme != "http" {
		t.Errorf("Unexpected URL: %v", f.Request.URL)
	}
}
