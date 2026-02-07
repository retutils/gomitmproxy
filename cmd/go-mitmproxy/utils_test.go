package main

import (
	"encoding/base64"
	"net/http/httptest"
	"testing"
)

func TestNewDefaultBasicAuth(t *testing.T) {
    // Valid
    auth, err := NewDefaultBasicAuth("user:pass|admin:secret")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if auth.Auth["user"] != "pass" {
        t.Error("user auth mismatch")
    }
    if auth.Auth["admin"] != "secret" {
        t.Error("admin auth mismatch")
    }
}

func TestEntryAuth(t *testing.T) {
    auth, err := NewDefaultBasicAuth("user:pass")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    // No header
    req := httptest.NewRequest("GET", "http://example.com", nil)
    res := httptest.NewRecorder()
    ok, err := auth.EntryAuth(res, req)
    if ok || err == nil {
        t.Error("expected missing auth error")
    }
    
    // Invalid header
    req.Header.Set("Proxy-Authorization", "Basic invalid")
    ok, err = auth.EntryAuth(res, req)
    if ok {
        t.Error("expected decode error/failure")
    }
    
    // Wrong creds
    cred := base64.StdEncoding.EncodeToString([]byte("user:wrong"))
    req.Header.Set("Proxy-Authorization", "Basic "+cred)
    ok, err = auth.EntryAuth(res, req)
    if ok {
        t.Error("expected invalid credentials")
    }
    
    // Correct creds
    cred = base64.StdEncoding.EncodeToString([]byte("user:pass"))
    req.Header.Set("Proxy-Authorization", "Basic "+cred)
    ok, err = auth.EntryAuth(res, req)
    if !ok || err != nil {
        t.Errorf("expected success, got err: %v", err)
    }
}
