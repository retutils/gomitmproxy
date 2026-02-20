package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestWebSocketMonitor(t *testing.T) {
	monitor := &WebSocketMonitor{}
	
	flow := &proxy.Flow{
		Request: &proxy.Request{
			Method: "GET",
			URL:    &url.URL{Scheme: "ws", Host: "example.com"},
		},
	}
	monitor.WebsocketHandshake(flow)
	
	msg := &proxy.WebSocketMessage{
		FromClient: true,
		Type:       1,
		Data:       []byte("hello"),
	}
	monitor.WebsocketMessage(flow, msg)
	
	msg.FromClient = false
	monitor.WebsocketMessage(flow, msg)
}

func TestRun(t *testing.T) {
	go Run()
	time.Sleep(100 * time.Millisecond)
}
