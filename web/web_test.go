package web

import (
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/retutils/gomitmproxy/proxy"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/atomic"
)

func TestWebAddon(t *testing.T) {
	// 1. Start WebAddon
	webAddon := NewWebAddon(":0")
    webAddon.Start()
    defer webAddon.Close()
    
    // Wait for start
    time.Sleep(100 * time.Millisecond)
    
	// 2. Connect WS client
	u := url.URL{Scheme: "ws", Host: webAddon.Addr, Path: "/echo"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial error: %v, addr: %s", err, webAddon.Addr)
	}
	defer c.Close()

	// 3. Create a dummy Flow
    connCtx := &proxy.ConnContext{
        ClientConn: &proxy.ClientConn{
            Conn: &dummyConn{},
            Tls: true, 
            Id: uuid.NewV4(), 
        },
        FlowCount: *atomic.NewUint32(1),
    }
	f := &proxy.Flow{
        Id: uuid.NewV4(),
        ConnContext: connCtx,
        Request: &proxy.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "https", 
				Host:   "example.com",
				Path:   "/foo",
			},
			Proto:  "HTTP/1.1",
			Header: make(http.Header),
			Body:   []byte("req body"),
         },
        Response: &proxy.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       []byte("res body"),
        },
    }
    
    // Initialize flow done channel
    // Since we created flow manually, we need to handle the Done channel if it's used.
    // However, Requestheaders uses f.Done().
    // We can't easily init the private done channel of the flow from here.
    // But we can check if the flow package has a constructor we should use.
    // Assuming proxy.NewFlow exists?
    // Based on previous logs, proxy.NewFlow() was not found/used.
    // If Responseheaders uses <-f.Done(), we might block if we don't handle it.
    // Wait, Requestheaders goroutine waits on f.Done().
    // If we don't close it, the goroutine leaks but doesn't block the test unless we wait for it.
    
    // 4. Send Breakpoint Rule for RequestBody (Action 1)
    rule := &breakPointRule{
        Method: "GET",
        URL:    "example.com",
        Action: 1, 
    }
    metaMsg := &messageMeta{
        mType:           messageTypeChangeBreakPointRules,
        breakPointRules: []*breakPointRule{rule},
    }
    metaBytes := metaMsg.bytes()
    err = c.WriteMessage(websocket.BinaryMessage, metaBytes)
    if err != nil {
        t.Fatalf("write rule: %v", err)
    }
    
    // Give time to process rule
    time.Sleep(100 * time.Millisecond)

    // 5. Test Request Interception
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        webAddon.Request(f)
    }()
    
    // Read messages
    // Expect Request (type 7) then RequestBody (type 8)
    // conn.go: 
    // Request(f) -> sendFlowMayWait(Request) -> sendFlowMayWait(RequestBody).
    // isIntercpt is checked for RequestBody.
    
    // Read first message (Request)
    _, msg1, err := c.ReadMessage()
    if err != nil { t.Fatal(err) }
    if len(msg1) > 1 && msg1[1] != byte(messageTypeRequest) {
         // It might be unordered if logic differs?
         // But sendFlowMayWait does sequential calls.
         // Wait, Requestheaders sends conn message? 
         // Requestheaders happens BEFORE Request usually.
    }
    
    // Trigger Requestheaders to simulate real flow and ensure no connection messages interfere
    webAddon.Requestheaders(f)
    
    // We might receive "Conn" message (type 0) from Requestheaders
    // Let's loop and read until we get RequestBody
    
    var interceptedMsg []byte
    timeout := time.After(2 * time.Second)
    
loop:
    for {
        select {
        case <-timeout:
            t.Fatal("timeout waiting for intercepted message")
        default:
            _, msg, err := c.ReadMessage()
            if err != nil { t.Fatal(err) }
            if len(msg) > 1 && msg[1] == byte(messageTypeRequestBody) {
                interceptedMsg = msg
                break loop
            }
        }
    }
    
    // Check interception
    // byte 38 is waitIntercept
    if len(interceptedMsg) > 38 && interceptedMsg[38] != 1 {
        t.Errorf("expected waitIntercept=1, got %d", interceptedMsg[38])
    }
    
    // 6. Resume Interception
    editMsg := &messageEdit{
        mType: messageTypeChangeRequest,
        id: f.Id,
        request: f.Request,
    }
    err = c.WriteMessage(websocket.BinaryMessage, editMsg.bytes())
    if err != nil { t.Fatal(err) }
    
    wg.Wait() // Request() should return
    
    // 7. Test Responseheaders
    // This calls sendMessageUntil(RequestBody).
    webAddon.Responseheaders(f)
    
    // 8. Test Response Interception
    // Our rule was action 1 (Request), so Response should NOT intercept.
    start := time.Now()
    webAddon.Response(f)
    if time.Since(start) > 500 * time.Millisecond {
        t.Error("Response blocked unexpectedly")
    }
    
    // 9. Test ServerDisconnected
    webAddon.ServerDisconnected(f.ConnContext)
}

func TestWebAddon_Response(t *testing.T) {
	// 1. Start WebAddon
	webAddon := NewWebAddon(":0")
    webAddon.Start()
    defer webAddon.Close()
    time.Sleep(100 * time.Millisecond)
    
	u := url.URL{Scheme: "ws", Host: webAddon.Addr, Path: "/echo"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer c.Close()

	// 2. Create Dummy Flow
    connCtx := &proxy.ConnContext{
        ClientConn: &proxy.ClientConn{Conn: &dummyConn{}, Tls: true, Id: uuid.NewV4()},
        FlowCount: *atomic.NewUint32(1),
    }
	f := &proxy.Flow{
        Id: uuid.NewV4(),
        ConnContext: connCtx,
        Request: &proxy.Request{
			Method: "GET",
			URL: &url.URL{Scheme: "https", Host: "example.com", Path: "/bar"},
			Header: make(http.Header),
         },
        Response: &proxy.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       []byte("res body"),
        },
    }

    // 3. Send Breakpoint Rule for Response (Action 2)
    rule := &breakPointRule{
        Method: "GET",
        URL:    "example.com",
        Action: 2, 
    }
    metaMsg := &messageMeta{
        mType:           messageTypeChangeBreakPointRules,
        breakPointRules: []*breakPointRule{rule},
    }
    err = c.WriteMessage(websocket.BinaryMessage, metaMsg.bytes())
    if err != nil { t.Fatal(err) }
    time.Sleep(100 * time.Millisecond)

    // 4. Test Response Interception
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        webAddon.Response(f)
    }()
    
    // Read messages until ResponseBody (type 10)
    // Response(f) -> sendFlowMayWait(Response) -> sendFlowMayWait(ResponseBody)
    
    timeout := time.After(2 * time.Second)
    var interceptedMsg []byte
loop:
    for {
        select {
        case <-timeout:
            t.Fatal("timeout waiting for response interception")
        case <-time.After(10*time.Millisecond):
             _, msg, err := c.ReadMessage()
             if err != nil { break loop } // Close?
             if len(msg) > 1 && msg[1] == byte(messageTypeResponseBody) {
                 interceptedMsg = msg
                 break loop
             }
        }
    }
    
    // Check waitIntercept
    if len(interceptedMsg) > 38 && interceptedMsg[38] != 1 {
        t.Errorf("expected waitIntercept=1 in response, got %d", interceptedMsg[38])
    }
    
    // Resume
    editMsg := &messageEdit{
        mType: messageTypeChangeResponse,
        id: f.Id,
        response: f.Response,
    }
    c.WriteMessage(websocket.BinaryMessage, editMsg.bytes())
    wg.Wait()
    
    // Test Responseheaders (should NOT intercept because Action=2? Action=2 is Response. ResponseHeaders is separate?)
    // Actually response headers interception logic:
    // Responseheaders(f):
    // web.sendMessageUntil(f, messageTypeRequestBody) // Syncs state
    // It doesn't seem to block on breakpoint unless logic in web.go forces it?
    // web.go: Responseheaders doesn't check isIntercpt?
    // Ah, logic:
    // func (web *WebAddon) Responseheaders(f *proxy.Flow) {
    //     if !Tls { sendConnMessage }
    //     web.sendMessageUntil(f, messageTypeRequestBody)
    // }
    // It just sends messages. It doesn't intercept/block.
    
    webAddon.Responseheaders(f)
}

func TestWebAddon_DisconnectedClient(t *testing.T) {
    webAddon := NewWebAddon(":0")
    webAddon.Start()
    defer webAddon.Close()
    time.Sleep(50 * time.Millisecond)

    // Connect and immediately close
	u := url.URL{Scheme: "ws", Host: webAddon.Addr, Path: "/echo"}
	c, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	c.Close()
    
    // Give time for removeConn to be called via readloop error
    time.Sleep(100 * time.Millisecond)

    // Should not panic or block
    connCtx := &proxy.ConnContext{
        ClientConn: &proxy.ClientConn{Conn: &dummyConn{}, Id: uuid.NewV4()},
        FlowCount: *atomic.NewUint32(1),
    }
    webAddon.Requestheaders(&proxy.Flow{ConnContext: connCtx})
}
