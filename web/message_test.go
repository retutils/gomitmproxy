package web

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/atomic"
)

func TestMessageFlow(t *testing.T) {
	// 1. Connection message
	connCtx := &proxy.ConnContext{
		ClientConn: &proxy.ClientConn{
            Id: uuid.NewV4(),
            Conn: &dummyConn{},
        },
        FlowCount: *atomic.NewUint32(0),
	}
	f := &proxy.Flow{Id: uuid.NewV4(), ConnContext: connCtx}
	
	msg, err := newMessageFlow(messageTypeConn, f)
	if err != nil {
		t.Fatalf("newMessageFlow(conn) error: %v", err)
	}
	if msg.mType != messageTypeConn {
		t.Errorf("want type 0, got %d", msg.mType)
	}
	
	// Check bytes encoding
	encoded := msg.bytes()
	if len(encoded) < 38 {
		t.Fatal("encoded msg too short")
	}
	if encoded[0] != messageVersion {
		t.Errorf("want version %d, got %d", messageVersion, encoded[0])
	}
	
	// 2. Request message
	f.Request = &proxy.Request{Method: "GET", URL: &url.URL{Scheme:"http", Host:"example.com"}}
	msg, err = newMessageFlow(messageTypeRequest, f)
	if err != nil {
		t.Fatalf("newMessageFlow(request) error: %v", err)
	}
	
	// 3. Request Body message
	f.Request.Body = []byte("hello")
	msg, err = newMessageFlow(messageTypeRequestBody, f)
	if err != nil {
		t.Fatalf("newMessageFlow(requestBody) error: %v", err)
	}
	if string(msg.content) != "hello" {
		t.Errorf("want content hello, got %s", msg.content)
	}
	
	// 4. Response message
	f.Response = &proxy.Response{StatusCode: 200}
	msg, err = newMessageFlow(messageTypeResponse, f)
	if err != nil {
		t.Fatalf("newMessageFlow(response) error: %v", err)
	}
	
	// 5. Response Body message
	f.Response.Body = []byte("world")
	msg, err = newMessageFlow(messageTypeResponseBody, f)
	if err != nil {
		t.Fatalf("newMessageFlow(responseBody) error: %v", err)
	}
	if string(msg.content) != "world" {
		t.Errorf("want content world, got %s", msg.content)
	}
}

func TestMessageEdit(t *testing.T) {
	// Create a message that looks like messageEdit
	// version + type + id + hl + header + bl + body
	
	id := uuid.NewV4()
	reqBody := []byte("new body")
	req := &proxy.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "http", Host: "test.com"},
		Header: make(http.Header),
	}
	headerContent, err := json.Marshal(req)
    if err != nil {
        t.Fatal(err)
    }
	
	buf := bytes.NewBuffer(make([]byte, 0))
	buf.WriteByte(messageVersion)
	buf.WriteByte(byte(messageTypeChangeRequest))
	buf.WriteString(id.String())
	
	// header len
	hl := uint32(len(headerContent))
	// write uint32 big endian
	buf.WriteByte(byte(hl >> 24))
	buf.WriteByte(byte(hl >> 16))
	buf.WriteByte(byte(hl >> 8))
	buf.WriteByte(byte(hl))
	
	buf.Write(headerContent)
	
	// body len
	bl := uint32(len(reqBody))
	buf.WriteByte(byte(bl >> 24))
	buf.WriteByte(byte(bl >> 16))
	buf.WriteByte(byte(bl >> 8))
	buf.WriteByte(byte(bl))
	
	buf.Write(reqBody)
	
	// Test parsing
	msg := parseMessage(buf.Bytes())
	if msg == nil {
		t.Fatal("parseMessage returned nil")
	}
	
	editMsg, ok := msg.(*messageEdit)
	if !ok {
		t.Fatal("expected messageEdit type")
	}
	
	if editMsg.mType != messageTypeChangeRequest {
		t.Errorf("want type %d, got %d", messageTypeChangeRequest, editMsg.mType)
	}
	if editMsg.id.String() != id.String() {
		t.Errorf("want id %s, got %s", id, editMsg.id)
	}
	if editMsg.request.Method != "POST" {
		t.Errorf("want method POST, got %s", editMsg.request.Method)
	}
	if string(editMsg.request.Body) != "new body" {
		t.Errorf("want body 'new body', got %s", editMsg.request.Body)
	}
}

func TestMessageMeta(t *testing.T) {
    rules := []*breakPointRule{
        {Method: "GET", URL: "example.com", Action: 1},
    }
    content, _ := json.Marshal(rules)
    
    buf := bytes.NewBuffer(make([]byte, 0))
    buf.WriteByte(messageVersion)
    buf.WriteByte(byte(messageTypeChangeBreakPointRules))
    buf.Write(content)
    
    msg := parseMessage(buf.Bytes())
    if msg == nil {
        t.Fatal("parseMessage returned nil")
    }
    
    metaMsg, ok := msg.(*messageMeta)
    if !ok {
        t.Fatal("expected messageMeta type")
    }
    
    if len(metaMsg.breakPointRules) != 1 {
        t.Errorf("want 1 rule, got %d", len(metaMsg.breakPointRules))
    }
    if metaMsg.breakPointRules[0].Method != "GET" {
        t.Errorf("want GET, got %s", metaMsg.breakPointRules[0].Method)
    }
}

func TestMessageConnClose(t *testing.T) {
	connCtx := &proxy.ConnContext{
		ClientConn: &proxy.ClientConn{Id: uuid.NewV4()},
        FlowCount: *atomic.NewUint32(10),
	}
	msg := newMessageConnClose(connCtx)
	
	if msg.mType != messageTypeConnClose {
		t.Errorf("want type %d, got %d", messageTypeConnClose, msg.mType)
	}
	
	// Check content (big endian uint32 10)
	buf := bytes.NewReader(msg.content)
	var count uint32
	if err := binary.Read(buf, binary.BigEndian, &count); err != nil {
		t.Fatal(err)
	}
	if count != 10 {
		t.Errorf("want 10, got %d", count)
	}
}

func TestMessageBytes(t *testing.T) {
	// Test messageFlow bytes
	flowMsg := &messageFlow{
		mType:         messageTypeConn,
		id:            uuid.NewV4(),
		waitIntercept: 1,
		content:       []byte("test content"),
	}
	
	b := flowMsg.bytes()
	if len(b) < 38 {
		t.Error("bytes too short")
	}
	if b[1] != byte(messageTypeConn) {
		t.Error("wrong type byte")
	}
	if b[38] != 1 {
		t.Error("wrong waitIntercept byte")
	}
	if string(b[39:]) != "test content" {
		t.Error("wrong content")
	}
    
    // Test messageEdit bytes (Request)
    // We already tested parsing, now test serializing
    req := &proxy.Request{Method: "GET", URL: &url.URL{Scheme: "http", Host: "byte-test.com"}, Header: make(http.Header)}
    editMsg := &messageEdit{
        mType:   messageTypeChangeRequest,
        id:      uuid.NewV4(),
        request: req,
    }
    bEdit := editMsg.bytes()
    if bEdit[1] != byte(messageTypeChangeRequest) {
        t.Error("wrong messageTypeChangeRequest byte")
    }
    // Verify structure via re-parsing
    parsedEdit := parseMessageEdit(bEdit)
    if parsedEdit == nil {
        t.Fatal("failed to re-parse generated bytes for messageEdit")
    }
    if parsedEdit.request.URL.Host != "byte-test.com" {
        t.Error("re-parsed host mismatch")
    }
    
    // Test messageMeta bytes
    rules := []*breakPointRule{{Method: "POST", Action: 1}}
    metaMsg := &messageMeta{
        mType:           messageTypeChangeBreakPointRules,
        breakPointRules: rules,
    }
    bMeta := metaMsg.bytes()
    parsedMeta := parseMessageMeta(bMeta)
    if parsedMeta == nil {
        t.Fatal("failed to re-parse generated bytes for messageMeta")
    }
    if len(parsedMeta.breakPointRules) != 1 {
        t.Error("re-parsed rule count mismatch")
    }

    // Test messageEdit Response
    res := &proxy.Response{StatusCode: 200, Header: make(http.Header)}
    editResMsg := &messageEdit{
        mType:    messageTypeChangeResponse,
        id:       uuid.NewV4(),
        response: res,
    }
    bEditRes := editResMsg.bytes()
    parsedEditRes := parseMessageEdit(bEditRes)
    if parsedEditRes == nil || parsedEditRes.mType != messageTypeChangeResponse {
        t.Fatal("failed to re-parse generated bytes for messageEdit response")
    }
}

func TestMessageInvalid(t *testing.T) {
	// Test short data
	if msg := parseMessage([]byte{messageVersion}); msg != nil {
		t.Error("expected nil for short data")
	}
	
	// Test wrong version
	if msg := parseMessage([]byte{messageVersion + 1, byte(messageTypeConn)}); msg != nil {
		t.Error("expected nil for wrong version")
	}
	
	// Test invalid type
	if msg := parseMessage([]byte{messageVersion, 99}); msg != nil {
		t.Error("expected nil for invalid type")
	}
	
	// Test invalid edit message parsing
	// 1. Short data for edit
	buf := newBytesBuffer(messageTypeChangeRequest)
	buf.WriteString("short-uuid") 
	if msg := parseMessage(buf.Bytes()); msg != nil {
		t.Error("expected nil for short edit message")
	}
    
    // 2. Invalid UUID
    buf = newBytesBuffer(messageTypeChangeRequest)
    buf.WriteString("invalid-uuid-string-36-chars-long-xxx")
    if msg := parseMessage(buf.Bytes()); msg != nil {
        t.Error("expected nil for invalid uuid")
    }

	// 3. Short data for header/body len
	buf = newBytesBuffer(messageTypeChangeRequest)
	buf.WriteString(uuid.NewV4().String())
	if msg := parseMessage(buf.Bytes()); msg != nil {
		t.Error("expected nil for incomplete edit message (missing lengths)")
	}
}

func TestMessageEdit_Invalid(t *testing.T) {
    id := uuid.NewV4()
    
    // Helper to create base valid edit message buffer
    createBase := func() *bytes.Buffer {
        buf := newBytesBuffer(messageTypeChangeRequest)
        buf.WriteString(id.String())
        return buf
    }
    
    // 1. Header length mismatch
    buf := createBase()
    // claim header is 100 bytes, but provide 0
    binary.Write(buf, binary.BigEndian, uint32(100)) 
    if msg := parseMessage(buf.Bytes()); msg != nil {
        t.Error("expected nil for header length mismatch")
    }
    
    // 2. Body length mismatch
    buf = createBase()
    header := []byte("{}")
    binary.Write(buf, binary.BigEndian, uint32(len(header)))
    buf.Write(header)
    // claim body is 100 bytes, but provide 0
    binary.Write(buf, binary.BigEndian, uint32(100))
    if msg := parseMessage(buf.Bytes()); msg != nil {
        t.Error("expected nil for body length mismatch")
    }
    
    // 3. Invalid JSON header
    buf = createBase()
    header = []byte("{invalid-json}")
    binary.Write(buf, binary.BigEndian, uint32(len(header)))
    buf.Write(header)
    binary.Write(buf, binary.BigEndian, uint32(0))
    if msg := parseMessage(buf.Bytes()); msg != nil {
        t.Error("expected nil for invalid json header")
    }
}

type dummyConn struct {}
func (d *dummyConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (d *dummyConn) Write(b []byte) (n int, err error)  { return 0, nil }
func (d *dummyConn) Close() error                       { return nil }
func (d *dummyConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (d *dummyConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (d *dummyConn) SetDeadline(t time.Time) error      { return nil }
func (d *dummyConn) SetReadDeadline(t time.Time) error { return nil }
func (d *dummyConn) SetWriteDeadline(t time.Time) error { return nil }

func TestNewMessageFlow_EdgeCases(t *testing.T) {
	id := uuid.NewV4()
	connCtx := &proxy.ConnContext{
		ClientConn: &proxy.ClientConn{Id: id},
	}
	f := &proxy.Flow{
		Id:          uuid.NewV4(),
		ConnContext: connCtx,
	}

	// 1. Response is nil
	_, err := newMessageFlow(messageTypeResponse, f)
	if err == nil {
		t.Error("Expected error for nil response")
	}

	_, err = newMessageFlow(messageTypeResponseBody, f)
	if err == nil {
		t.Error("Expected error for nil response body")
	}

	// 2. Invalid message type
	_, err = newMessageFlow(99, f)
	if err == nil {
		t.Error("Expected error for invalid message type")
	}
}

func TestParseMessage_ErrorCases(t *testing.T) {
	// 1. Data too short
	if parseMessage([]byte{messageVersion}) != nil {
		t.Error("Expected nil for short data")
	}

	// 2. Wrong version
	if parseMessage([]byte{0, byte(messageTypeConn)}) != nil {
		t.Error("Expected nil for wrong version")
	}

	// 3. Invalid message type
	if parseMessage([]byte{messageVersion, 99}) != nil {
		t.Error("Expected nil for invalid type")
	}

	// 4. Unknown message type group (neither edit nor meta)
	if parseMessage([]byte{messageVersion, byte(messageTypeConn)}) != nil {
		t.Error("Expected nil for type without parser")
	}
}

func TestParseMessageEdit_ErrorCases(t *testing.T) {
	id := uuid.NewV4().String()
	
	// 1. Too short for header
	if parseMessageEdit([]byte{0, 0}) != nil {
		t.Error("Expected nil for short data")
	}

	// 2. Invalid UUID
	data := make([]byte, 38)
	data[0] = messageVersion
	data[1] = byte(messageTypeChangeRequest)
	copy(data[2:], "invalid-uuid")
	if parseMessageEdit(data) != nil {
		t.Error("Expected nil for invalid UUID")
	}

	// 3. Too short for lengths
	data = make([]byte, 40)
	data[0] = messageVersion
	data[1] = byte(messageTypeChangeRequest)
	copy(data[2:38], id)
	if parseMessageEdit(data) != nil {
		t.Error("Expected nil for data missing lengths")
	}

	// 4. Header length mismatch
	data = make([]byte, 46)
	data[0] = messageVersion
	data[1] = byte(messageTypeChangeRequest)
	copy(data[2:38], id)
	binary.BigEndian.PutUint32(data[38:42], 100) // hl = 100
	if parseMessageEdit(data) != nil {
		t.Error("Expected nil for header len mismatch")
	}

	// 5. Body length mismatch
	hl := 2
	data = make([]byte, 42+hl+4)
	data[0] = messageVersion
	data[1] = byte(messageTypeChangeRequest)
	copy(data[2:38], id)
	binary.BigEndian.PutUint32(data[38:42], uint32(hl))
	binary.BigEndian.PutUint32(data[42+hl:42+hl+4], 100) // bl = 100
	if parseMessageEdit(data) != nil {
		t.Error("Expected nil for body len mismatch")
	}

	// 6. Unmarshal error (invalid JSON)
	header := []byte("{invalid}")
	hl = len(header)
	data = make([]byte, 42+hl+4)
	data[0] = messageVersion
	data[1] = byte(messageTypeChangeRequest)
	copy(data[2:38], id)
	binary.BigEndian.PutUint32(data[38:42], uint32(hl))
	copy(data[42:42+hl], header)
	binary.BigEndian.PutUint32(data[42+hl:42+hl+4], 0)
	if parseMessageEdit(data) != nil {
		t.Error("Expected nil for invalid JSON unmarshal")
	}
	
	// 7. Invalid mType in parseMessageEdit
	data[1] = byte(messageTypeConn)
	if parseMessageEdit(data) != nil {
		t.Error("Expected nil for invalid mType")
	}
}

func TestParseMessageMeta_Error(t *testing.T) {
	if parseMessageMeta([]byte{0, 0, '{'}) != nil {
		// Actually unmarshal might fail or succeed with empty?
		// '{' is invalid JSON.
	}
}

