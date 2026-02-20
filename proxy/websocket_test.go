package proxy

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// MockAddon implements Addon interface to capture events
type MockAddon struct {
	BaseAddon
	Handshakes []string
	Messages   []*WebSocketMessage
	mu         sync.Mutex
}

func (m *MockAddon) WebsocketHandshake(f *Flow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Handshakes = append(m.Handshakes, f.Request.URL.String())
}

func (m *MockAddon) WebsocketMessage(f *Flow, msg *WebSocketMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Deep copy msg because Data buffer might be reused or modified
	dataCopy := make([]byte, len(msg.Data))
	copy(dataCopy, msg.Data)
	m.Messages = append(m.Messages, &WebSocketMessage{
		Type:       msg.Type,
		Data:       dataCopy,
		FromClient: msg.FromClient,
	})
}

func TestWebSocket_Integration(t *testing.T) {
	logrus.SetOutput(io.Discard) // Quiet logs

	// 1. Start Backend WebSocket Echo Server
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	addons := []Addon{}
	mockAddon := &MockAddon{}
	addons = append(addons, mockAddon)

	// 2. Start Proxy Server
	// We need a handler that simulates the proxy logic invoking wss
	proxyServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock ID injection usually done by Proxy
		ctx := context.WithValue(r.Context(), connContextKey, &ConnContext{ClientConn: &ClientConn{}})
		r = r.WithContext(ctx)
		
		// Force the target host to be our backend server
		// In real mitmproxy, this comes from the CONNECT request or Host header
		r.Host = backendURL.Host
		r.URL.Scheme = "https"
		r.URL.Host = backendURL.Host
		
		ws := &webSocket{}
		// Trust the backend cert
		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		
		ws.wss(w, r, tlsConfig, addons)
	}))
	defer proxyServer.Close()

	// 3. Client connects to Proxy
	// We need to Connect to proxyServer.URL but telling it to upgrade.
	// We also need to trust proxyServer's cert.
	proxyURL, _ := url.Parse(proxyServer.URL)
	proxyWSURL := "wss://" + proxyURL.Host + "/ws"

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	
	c, _, err := dialer.Dial(proxyWSURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	// 4. Send Message
	message := []byte("hello world")
	err = c.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// 5. Read Message (Echo)
	_, recv, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(recv) != string(message) {
		t.Errorf("expected %s, got %s", message, recv)
	}

	// 6. Verify Addon Events
	// Note: Message interception happens asynchronously, so we might need to wait a tiny bit
	// But since we received the echo, the server->client message must have passed through the proxy loop.
	// However, the proxy writes to client and THEN loop continues, so the Addon hook calls happen before Write.
	// So it should be recorded.
	
	// Wait a moment for async processing (just in case of race in test assertion)
	time.Sleep(100 * time.Millisecond)

	mockAddon.mu.Lock()
	defer mockAddon.mu.Unlock()

	if len(mockAddon.Handshakes) == 0 {
		t.Error("expected handshake event")
	}

	if len(mockAddon.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(mockAddon.Messages))
	} else {
		// Verify Client -> Server
		msg1 := mockAddon.Messages[0]
		if !msg1.FromClient {
			t.Error("first message should be from client")
		}
		if string(msg1.Data) != string(message) {
			t.Errorf("expected msg1 data %s, got %s", message, msg1.Data)
		}
		
		// Verify Server -> Client
		msg2 := mockAddon.Messages[1]
		if msg2.FromClient {
			t.Error("second message should be from server")
		}
		if string(msg2.Data) != string(message) {
			t.Errorf("expected msg2 data %s, got %s", message, msg2.Data)
		}
	}
}

func TestWebSocket_DialFail(t *testing.T) {
	ws := &webSocket{}
	rec := httptest.NewRecorder()
	u, _ := url.Parse("http://invalid-host.local")
	req := httptest.NewRequest("GET", u.String(), nil)
	req.Host = u.Host
	ctx := context.WithValue(req.Context(), connContextKey, &ConnContext{ClientConn: &ClientConn{}})
	req = req.WithContext(ctx)
	
	ws.wss(rec, req, nil, nil)
	if rec.Code != 502 {
		t.Errorf("Expected 502, got %d", rec.Code)
	}
}

func TestWebSocket_UpgradeFail(t *testing.T) {
	// Mock server that returns a 200 but NOT a switching protocol
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	ws := &webSocket{}
	rec := httptest.NewRecorder()
	u, _ := url.Parse(backend.URL)
	req := httptest.NewRequest("GET", u.String(), nil)
	req.Host = u.Host
	ctx := context.WithValue(req.Context(), connContextKey, &ConnContext{ClientConn: &ClientConn{}})
	req = req.WithContext(ctx)
	
	ws.wss(rec, req, nil, nil)
	// It should fail dial because it's not a websocket server
}
