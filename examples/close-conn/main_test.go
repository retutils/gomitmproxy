package main

import (
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestCloseConn(t *testing.T) {
	addon := &CloseConn{}
	
	client := &proxy.ClientConn{}
	addon.ClientConnected(client)
	if client.UpstreamCert {
		t.Error("Expected UpstreamCert false")
	}
	
	flow := &proxy.Flow{}
	addon.Requestheaders(flow)
	if flow.Response == nil || flow.Response.StatusCode != 502 {
		t.Error("Expected mocked 502 response")
	}
}

func TestRun(t *testing.T) {
	go Run()
	time.Sleep(100 * time.Millisecond)
}
