package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestRewriteHost(t *testing.T) {
	addon := &RewriteHost{}
	
	client := &proxy.ClientConn{}
	addon.ClientConnected(client)
	if client.UpstreamCert {
		t.Error("Expected UpstreamCert false")
	}
	
	flow := &proxy.Flow{
		Request: &proxy.Request{
			Method: "GET",
			URL:    &url.URL{Scheme: "https", Host: "example.com"},
		},
	}
	addon.Requestheaders(flow)
	if flow.Request.URL.Host != "www.baidu.com" {
		t.Errorf("Expected host rewrite to baidu, got %s", flow.Request.URL.Host)
	}
}

func TestRun(t *testing.T) {
	go Run()
	time.Sleep(100 * time.Millisecond)
}
