package proxy

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// 当前仅做了转发 websocket 流量

type webSocket struct{}

var defaultWebSocket webSocket

func (s *webSocket) wss(res http.ResponseWriter, req *http.Request, tlsConfig *tls.Config, addons []Addon) {
	log := log.WithField("in", "webSocket.wss").WithField("host", req.Host)

	f := NewFlow()
	f.Request = NewRequest(req)
	f.ConnContext = req.Context().Value(connContextKey).(*ConnContext)
	defer f.Finish()

	// 1. Dial backend
	host := req.Host
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}
	targetURL := url.URL{Scheme: "wss", Host: host, Path: req.URL.Path, RawQuery: req.URL.RawQuery}

	// Copy headers
	requestHeader := http.Header{}
	for k, v := range req.Header {
		if k == "Upgrade" || k == "Connection" || k == "Sec-Websocket-Key" || k == "Sec-Websocket-Version" || k == "Sec-Websocket-Extensions" {
			continue
		}
		requestHeader[k] = v
	}

	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
	}
	serverConn, resp, err := dialer.Dial(targetURL.String(), requestHeader)
	if err != nil {
		log.Errorf("websocket dial: %v\n", err)
		res.WriteHeader(502)
		return
	}
	defer serverConn.Close()

	// 2. Upgrade client connection
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	if protocol := resp.Header.Get("Sec-WebSocket-Protocol"); protocol != "" {
		upgrader.Subprotocols = []string{protocol}
	}

	clientConn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Errorf("websocket upgrade: %v\n", err)
		return
	}
	// serverConn.Response is the response from the server. constructing response flow
	f.Response = &Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
	}

	for _, addon := range addons {
		addon.WebsocketHandshake(f)
	}

	defer clientConn.Close()

	// 3. Transfer loop
	errChan := make(chan error, 2)
	
	// Client -> Server
	go func() {
		for {
			messageType, p, err := clientConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			// Addon Interception
			msg := &WebSocketMessage{Type: messageType, Data: p, FromClient: true}
			for _, addon := range addons {
				addon.WebsocketMessage(f, msg)
			}
			
			if err := serverConn.WriteMessage(msg.Type, msg.Data); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Server -> Client
	go func() {
		for {
			messageType, p, err := serverConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			// Addon Interception
			msg := &WebSocketMessage{Type: messageType, Data: p, FromClient: false}
			for _, addon := range addons {
				addon.WebsocketMessage(f, msg)
			}
				
			if err := clientConn.WriteMessage(msg.Type, msg.Data); err != nil {
				errChan <- err
				return
			}
		}
	}()
	
	select {
	case err := <-errChan:
		log.Debugf("websocket loop end: %v", err)
	}
}
