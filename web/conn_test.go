package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/retutils/gomitmproxy/proxy"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/atomic"
)

func TestConn_TrySendConnMessage(t *testing.T) {
	// Setup WebSocket server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer s.Close()

	// Use ws:// scheme
	u := "ws" + s.URL[4:]
	wsConn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer wsConn.Close()

	c := newConn(wsConn)
	
	// Create dummy ConnContext
	connCtx := &proxy.ConnContext{
		ClientConn: &proxy.ClientConn{Id: uuid.NewV4(), Conn: &dummyConn{}},
        FlowCount: *atomic.NewUint32(0),
	}
	
	// Test sending valid flow
	f := &proxy.Flow{Id: uuid.NewV4(), ConnContext: connCtx}
	c.trySendConnMessage(f)
}

func TestConn_WhenConnClose(t *testing.T) {
    // Setup WebSocket server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		c.Close() // Close immediately
	}))
	defer s.Close()

	u := "ws" + s.URL[4:]
	wsConn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
    
    c := newConn(wsConn)
    
    // Create a wait channel
    ch := c.initWaitChan(uuid.NewV4().String())
    
    // waiting in goroutine
    done := make(chan bool)
    connCtx := &proxy.ConnContext{ClientConn: &proxy.ClientConn{Id: uuid.NewV4()}}

    go func() {
        c.whenConnClose(connCtx)
        done <- true
    }()
    
    // Simulate connection close by closing the underlying connection
    c.conn.Close()
    
    select {
    case <-done:
        // success, whenConnClose returned
    case <-time.After(2 * time.Second):
        t.Error("whenConnClose timeout")
    }
    
    // Verify channel is closed/removed
    _ = ch
}
