package proxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockHookAddon structure to inject behavior
type MockHookAddon struct {
	BaseAddon
	OnRequestheaders      func(*Flow)
	OnRequest             func(*Flow)
	OnResponseheaders     func(*Flow)
	OnResponse            func(*Flow)
	OnStreamRequestMod    func(*Flow, io.Reader) io.Reader
	OnStreamResponseMod   func(*Flow, io.Reader) io.Reader
}

func (m *MockHookAddon) Requestheaders(f *Flow) {
	if m.OnRequestheaders != nil {
		m.OnRequestheaders(f)
	}
}

func (m *MockHookAddon) Request(f *Flow) {
	if m.OnRequest != nil {
		m.OnRequest(f)
	}
}

func (m *MockHookAddon) Responseheaders(f *Flow) {
	if m.OnResponseheaders != nil {
		m.OnResponseheaders(f)
	}
}

func (m *MockHookAddon) Response(f *Flow) {
	if m.OnResponse != nil {
		m.OnResponse(f)
	}
}

func (m *MockHookAddon) StreamRequestModifier(f *Flow, in io.Reader) io.Reader {
	if m.OnStreamRequestMod != nil {
		return m.OnStreamRequestMod(f, in)
	}
	return in
}

func (m *MockHookAddon) StreamResponseModifier(f *Flow, in io.Reader) io.Reader {
	if m.OnStreamResponseMod != nil {
		return m.OnStreamResponseMod(f, in)
	}
	return in
}

// MockReader to simulate read errors
type MockReader struct {
	Data []byte
	Err  error
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	if m.Err != nil {
		return 0, m.Err
	}
	if len(m.Data) == 0 {
		return 0, io.EOF
	}
	n = copy(p, m.Data)
	m.Data = m.Data[n:]
	return n, nil
}

func TestAttacker_Attack_AddonPanic(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	
	// Add mock addon that panics
	mockAddon := &MockHookAddon{
		OnRequestheaders: func(f *Flow) {
			panic("simulating addon panic")
		},
	}
	p.AddAddon(mockAddon)

	a, _ := newAttacker(p) // Recreate attacker with addon

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	connCtx := &ConnContext{proxy: p}
	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	// Capture logs to verify panic recovery or just ensure it doesn't crash test
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("attack should recover from panic but didn't: %v", r)
		}
	}()

	a.attack(rec, req)
	// If we reached here, panic was recovered
}

func TestAttacker_Attack_RequestInterception(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	
	mockAddon := &MockHookAddon{
		OnRequestheaders: func(f *Flow) {
			f.Response = &Response{
				StatusCode: 200,
				Body:       []byte("intercepted"),
			}
		},
	}
	p.AddAddon(mockAddon)
	a, _ := newAttacker(p)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	connCtx := &ConnContext{proxy: p}
	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	a.attack(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "intercepted" {
		t.Errorf("Expected body 'intercepted', got '%s'", rec.Body.String())
	}
}

func TestAttacker_Attack_DialError(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	a, _ := newAttacker(p)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	connCtx := &ConnContext{proxy: p}
	// Inject mock dial function that fails
	connCtx.dialFn = func(ctx context.Context) error {
		return errors.New("connection failed")
	}

	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	a.attack(rec, req)

	if rec.Code != 502 {
		t.Errorf("Expected status 502 for dial error, got %d", rec.Code)
	}
}

func TestAttacker_Attack_ProxyAuthError(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	a, _ := newAttacker(p)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	connCtx := &ConnContext{proxy: p}
	connCtx.dialFn = func(ctx context.Context) error {
		return errors.New("Proxy Authentication Required")
	}

	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	a.attack(rec, req)

	if rec.Code != 407 {
		t.Errorf("Expected status 407 (Proxy Auth Required), got %d", rec.Code)
	}
}

func TestAttacker_Attack_StreamLargeBody(t *testing.T) {
	opts := &Options{Addr: ":0", StreamLargeBodies: 1} // 1 byte limit
	p, _ := NewProxy(opts)
	a, _ := newAttacker(p)

	// Body larger than 1 byte
	reqBody := bytes.NewReader([]byte("large body"))
	req := httptest.NewRequest(http.MethodPost, "http://example.com", reqBody)
	connCtx := &ConnContext{proxy: p}
	
	// Mock successful dial to avoid actual network call
	connCtx.dialFn = func(ctx context.Context) error {
		// Mock server connection with a client that mocks response
		// This is getting complicated to mock http.Client.Do inside attack
		// We can use UseSeparateClient logic to use a.client, but we still need to mock Transport
		return nil
	}
	// Create a dummy server conn to bypass dialFn call if we wanted
	// But we'll rely on separate client or just let it fail later.
	// Actually we want to verify f.Stream is set.
	
	// To verify f.Stream, we can use an addon to inspect the Flow!
	
	var streamSet bool
	mockAddon := &MockHookAddon{
		OnRequestheaders: func(f *Flow) {
			// too early
		},
		OnStreamRequestMod: func(f *Flow, r io.Reader) io.Reader {
			streamSet = f.Stream
			return r
		},
	}
	p.AddAddon(mockAddon)
	a, _ = newAttacker(p) // Update a with addon

	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	// We expect 502 essentially because real network fails, but we want to check streamSet
	rec := httptest.NewRecorder()
	a.attack(rec, req)
	
	if !streamSet {
		t.Error("Expected f.Stream to be true for large body")
	}
}

func TestAttacker_Attack_RequestBodyReadError(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	a, _ := newAttacker(p)

	mockReader := &MockReader{Err: errors.New("read error")}
	req := httptest.NewRequest("POST", "http://example.com", mockReader)
	connCtx := &ConnContext{proxy: p}
	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	a.attack(rec, req)

	if rec.Code != 502 {
		t.Errorf("Expected 502 on request body read error, got %d", rec.Code)
	}
}

func TestAttacker_ServeHTTP_Normalization(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	a, _ := newAttacker(p)

	// Create request with missing Scheme and Host (common in raw HTTP handlers or proxy request parsing)
	req := httptest.NewRequest(http.MethodGet, "/path", nil)
	req.URL.Scheme = ""
	req.URL.Host = ""
	req.Host = "example.com" // Set Host header/property separately

	// We need to inject a dial function or mock connection context because ServeHTTP calls attack -> getUpstreamConn
	connCtx := &ConnContext{proxy: p}
	// Mock successful dial to avoid actual network call
	connCtx.dialFn = func(ctx context.Context) error {
		return nil
	}
	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	
	// We expect attack to be called. attack might fail due to no backend, but ServeHTTP logic should run.
	// We just want to ensure Scheme and Host are set.
	// Since we can't inspect req after modification easily without hooking attack, 
	// we will rely on coverage count or use a panic hook in attack if we could.
	// But simply running it covers the lines.
	a.ServeHTTP(rec, req)

	if req.URL.Scheme != "https" {
		t.Errorf("Expected Scheme https, got %s", req.URL.Scheme)
	}
	if req.URL.Host != "example.com" {
		t.Errorf("Expected Host example.com, got %s", req.URL.Host)
	}
}

func TestAttacker_Attack_Interception_Request(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	
	mockAddon := &MockHookAddon{
		OnRequest: func(f *Flow) {
			f.Response = &Response{
				StatusCode: 201,
				Body:       []byte("intercepted-req"),
			}
		},
	}
	p.AddAddon(mockAddon)
	a, _ := newAttacker(p)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	connCtx := &ConnContext{proxy: p}
	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	a.attack(rec, req)

	if rec.Code != 201 {
		t.Errorf("Expected 201, got %d", rec.Code)
	}
	if rec.Body.String() != "intercepted-req" {
		t.Errorf("Expected body 'intercepted-req', got '%s'", rec.Body.String())
	}
}

func TestAttacker_Attack_Interception_ResponseHeaders(t *testing.T) {
	opts := &Options{Addr: ":0"}
	p, _ := NewProxy(opts)
	
	mockAddon := &MockHookAddon{
		OnResponseheaders: func(f *Flow) {
			// Early reply
			f.Response.Body = []byte("intercepted-res-headers")
			// response already exists (from backend dial mock)
		},
	}
	p.AddAddon(mockAddon)
	a, _ := newAttacker(p) // Recreate attacker

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	connCtx := &ConnContext{proxy: p}
	
	// Mock successful backend dial
	connCtx.dialFn = func(ctx context.Context) error {
		return nil 
	}
	// We need ServerConn client to return something
	// Mock ServerConn with mock client is hard without replacing transport.
	// But we can rely on UseSeparateClient or something?
	// If dialFn succeeds but no ServerConn, it creates one?
	// But ServerConn.client uses real transport.
	// We need to Mock the Transport or Client.
	
	// Better approach: Mock connCtx.ServerConn pre-populated with mock client logic?
	// attacker logic:
	// proxyRes, err = f.ConnContext.ServerConn.client.Do(proxyReq)
	
	// We can replace ServerConn.client.Transport!
	serverConn := newServerConn()
	serverConn.client = &http.Client{
		Transport: &MockTransport{
			Response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("original")),
				Header:     make(http.Header),
			},
		},
	}
	connCtx.ServerConn = serverConn

	ctx := context.WithValue(req.Context(), connContextKey, connCtx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	a.attack(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
	// Responseheaders hook sets body to "intercepted-res-headers" and RETURNS EARLY (if logic holds)
	// Check attacker.go logic:
	// addon.Responseheaders(f)
	// if f.Response.Body != nil { a.reply(...); return }
	
	if rec.Body.String() != "intercepted-res-headers" {
		t.Errorf("Expected 'intercepted-res-headers', got '%s'", rec.Body.String())
	}
}

// MockTransport allows mocking http.Client execution
type MockTransport struct {
	Response *http.Response
	Err      error
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Response, nil
}

