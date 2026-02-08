package proxy

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebSocket_DialError(t *testing.T) {
	// Call wss with an invalid host to force Dial error
	req := httptest.NewRequest("GET", "http://invalid-host.local", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	
	ctx := context.WithValue(req.Context(), connContextKey, &ConnContext{
		ClientConn: &ClientConn{},
	})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	
	ws := &webSocket{}
	ws.wss(rec, req, &tls.Config{InsecureSkipVerify: true}, nil)

	if rec.Code != 502 {
		t.Errorf("Expected 502 status code for dial error, got %d", rec.Code)
	}
}

func TestWebSocket_UpgradeError(t *testing.T) {
	// Start a dummy backend to ensure Dial succeeds
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do nothing, just accept connection
	}))
	defer backend.Close()

	req := httptest.NewRequest("GET", backend.URL, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	
	ctx := context.WithValue(req.Context(), connContextKey, &ConnContext{
		ClientConn: &ClientConn{},
	})
	req = req.WithContext(ctx)

	// httptest.NewRecorder() does NOT implement http.Hijacker, so Upgrade will fail
	rec := httptest.NewRecorder()

	ws := &webSocket{}
	// Use InsecureSkipVerify to trust the test backend cert
	ws.wss(rec, req, &tls.Config{InsecureSkipVerify: true}, nil)

	// We expect Upgrade to fail. 
	// The function should log error and return. 
	// Since rec is not hijacked, we can check basic properties or just ensure no panic.
	// WSS doesn't write error status on upgrade failure, just logs and returns.
	
	// If it didn't panic, we assume it handled the error gracefully.
}
