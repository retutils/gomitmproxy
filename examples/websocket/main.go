package main

import (
	"encoding/hex"
	
	log "github.com/sirupsen/logrus"

	"github.com/retutils/gomitmproxy/proxy"
)

type WebSocketMonitor struct {
	proxy.BaseAddon
}

func (a *WebSocketMonitor) WebsocketHandshake(f *proxy.Flow) {
	log.Infof("WebSocket Handshake: %v %v", f.Request.Method, f.Request.URL.String())
}

func (a *WebSocketMonitor) WebsocketMessage(f *proxy.Flow, msg *proxy.WebSocketMessage) {
	direction := "Server->Client"
	if msg.FromClient {
		direction = "Client->Server"
	}
	
	log.Infof("WebSocket Message [%s]: Type=%d Len=%d Data=%s", direction, msg.Type, len(msg.Data), hex.EncodeToString(msg.Data))
}

func main() {
	opts := &proxy.Options{
		Addr:              ":9080",
		StreamLargeBodies: 1024 * 1024 * 5,
		SslInsecure:       true,
	}

	p, err := proxy.NewProxy(opts)
	if err != nil {
		log.Fatal(err)
	}

	p.AddAddon(&WebSocketMonitor{})

	log.Fatal(p.Start())
}
