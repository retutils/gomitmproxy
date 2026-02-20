package web

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/retutils/gomitmproxy/proxy"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/atomic"
)

func TestWebAddon_CoverageExtra(t *testing.T) {
	webAddon := NewWebAddon(":0")
	webAddon.Start()
	defer webAddon.Close()
	time.Sleep(100 * time.Millisecond)

	u := url.URL{Scheme: "ws", Host: webAddon.Addr, Path: "/echo"}
	c, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	time.Sleep(50 * time.Millisecond)

	// Get the concurrentConn created by WebAddon
	webAddon.connsMu.RLock()
	realWebConn := webAddon.conns[0]
	webAddon.connsMu.RUnlock()

	id := uuid.NewV4()
	connCtx := &proxy.ConnContext{
		ClientConn: &proxy.ClientConn{
			Conn: &dummyConn{},
			Tls:  false, // Non-TLS
			Id:   id,
		},
		FlowCount: *atomic.NewUint32(1),
	}
	f := &proxy.Flow{
		Id:          uuid.NewV4(),
		ConnContext: connCtx,
		Request: &proxy.Request{
			Method: "GET",
			URL:    &url.URL{Scheme: "http", Host: "example.com"},
		},
	}

	// 1. Responseheaders with !Tls
	webAddon.Responseheaders(f)
	webAddon.Responseheaders(f) // hits map check

	// 2. Hits error paths by closing the connection
	c.Close()
	time.Sleep(50 * time.Millisecond)

	// Now WriteMessage should fail
	realWebConn.trySendConnMessage(f)
	realWebConn.whenConnClose(connCtx)
	
	msg, _ := newMessageFlow(messageTypeRequest, f)
	realWebConn.writeMessage(msg)
	realWebConn.writeMessageMayWait(msg, f)
	
	// 3. sendMessageUntil with existing state
	webAddon.flowMu.Lock()
	webAddon.flowMessageState[f] = messageTypeResponseBody
	webAddon.flowMu.Unlock()
	webAddon.sendMessageUntil(f, messageTypeResponseBody)
}

func TestWebAddon_Start_Fail(t *testing.T) {
	// Use an invalid address or one that's already in use
	webAddon := NewWebAddon("invalid-addr:999999")
	webAddon.Start()
	// Should log error but not panic
}

func TestWebAddon_SendFlow_Error(t *testing.T) {
	webAddon := NewWebAddon(":0")
	// Add a connection
	webAddon.addConn(&concurrentConn{})
	
	// sendFlow with failing msgFn
	webAddon.sendFlow(func() (*messageFlow, error) {
		return nil, fmt.Errorf("gen error")
	})
	
	// sendFlowMayWait with failing msgFn
	webAddon.sendFlowMayWait(nil, func() (*messageFlow, error) {
		return nil, fmt.Errorf("gen error")
	})
}

func TestWebAddon_ReadLoop_Coverage(t *testing.T) {
	webAddon := NewWebAddon(":0")
	webAddon.Start()
	defer webAddon.Close()
	time.Sleep(100 * time.Millisecond)

	u := url.URL{Scheme: "ws", Host: webAddon.Addr, Path: "/echo"}
	c, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	defer c.Close()

	// 1. Not BinaryMessage
	c.WriteMessage(websocket.TextMessage, []byte("hello"))
	
	// 2. ParseMessage error (invalid bytes)
	c.WriteMessage(websocket.BinaryMessage, []byte{0xff, 0x00})
	
	// 3. MessageMeta (Breakpoint rules)
	metaMsg := &messageMeta{
		mType:           messageTypeChangeBreakPointRules,
		breakPointRules: []*breakPointRule{{Method: "GET", URL: "foo", Action: 1}},
	}
	c.WriteMessage(websocket.BinaryMessage, metaMsg.bytes())
	
	// 4. Invalid message type (e.g. unknown type in parseMessage but parseMessage returns nil)
	// parseMessage in message.go:
	// if !validMessageType(mType) { return nil }
	c.WriteMessage(websocket.BinaryMessage, []byte{0x00, 0x55}) // 0x55 is unknown
	
	time.Sleep(200 * time.Millisecond)
}

func TestWebAddon_IsIntercept_Coverage(t *testing.T) {
	c := &concurrentConn{
		breakPointRules: []*breakPointRule{
			{URL: "foo", Action: 3}, // both
			{Method: "POST", URL: "bar", Action: 1}, // request only
			{URL: "", Action: 3}, // empty URL (skip)
		},
	}
	
	f := &proxy.Flow{
		Request: &proxy.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/foo/bar"},
		},
	}
	
	// Hits rule 1
	if !c.isIntercpt(f, messageTypeRequestBody) {
		t.Error("expected intercept")
	}
	
	// Hits rule 2
	f.Request.Method = "POST"
	f.Request.URL.Path = "/bar"
	if !c.isIntercpt(f, messageTypeRequestBody) {
		t.Error("expected intercept")
	}
	
	// Wrong method for rule 2
	f.Request.Method = "GET"
	if c.isIntercpt(f, messageTypeRequestBody) {
		t.Error("expected NO intercept")
	}
}

func TestWebAddon_WaitIntercept_Drop(t *testing.T) {
	c := newConn(nil)
	f := &proxy.Flow{Id: uuid.NewV4()}
	
	ch := c.initWaitChan(f.Id.String())
	go func() {
		ch <- &messageEdit{mType: messageTypeDropRequest}
	}()
	
	c.waitIntercept(f)
	if f.Response == nil || f.Response.StatusCode != 502 {
		t.Error("expected drop response")
	}
}

func TestWebAddon_WaitIntercept_ChangeResponse(t *testing.T) {
	c := newConn(nil)
	f := &proxy.Flow{Id: uuid.NewV4(), Response: &proxy.Response{StatusCode: 200}}
	
	ch := c.initWaitChan(f.Id.String())
	go func() {
		ch <- &messageEdit{
			mType: messageTypeChangeResponse,
			response: &proxy.Response{
				StatusCode: 201,
				Header:     http.Header{"X-Test": []string{"foo"}},
				Body:       []byte("new body"),
			},
		}
	}()
	
	c.waitIntercept(f)
	if f.Response.StatusCode != 201 {
		t.Errorf("expected 201, got %d", f.Response.StatusCode)
	}
	if string(f.Response.Body) != "new body" {
		t.Error("body not changed")
	}
}
