package storage

import (
	"net/http"
	"net/url"
	"testing"
	
	"github.com/retutils/gomitmproxy/proxy"
	uuid "github.com/satori/go.uuid"
)

func TestService_SaveAndSearch(t *testing.T) {
	tmpDir := t.TempDir()
	
	svc, err := NewService(tmpDir)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	svc.Close() // Explicitly close to release lock

	// Test NewService idempotency (re-open existing)
	svc2, err := NewService(tmpDir)
	if err != nil {
		t.Fatalf("NewService re-open failed: %v", err)
	}
	svc = svc2 // Use svc2 for rest of test
	defer svc.Close()


	flowID := uuid.NewV4()
	flow := &proxy.Flow{
		Id: flowID,
		ConnContext: &proxy.ConnContext{
			ClientConn: &proxy.ClientConn{},
		},
		Request: &proxy.Request{
			Method: "POST",
			URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/test"},
			Proto:  "HTTP/1.1",
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"User-Agent":   []string{"Go-Test-Client"},
			},
			Body: []byte(`{"key": "value"}`),
		},
		Response: &proxy.Response{
			StatusCode: 201,
			Header: http.Header{
				"Server": []string{"UnitTestServer"},
			},
			Body: []byte(`{"status": "created"}`),
		},
	}

	// Test Save
	if err := svc.Save(flow); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Test Search by URL
	results, err := svc.Search("example.com")
	if err != nil {
		t.Fatalf("Search URL failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for URL search, got %d", len(results))
	} else {
		if results[0].ID != flowID.String() {
			t.Errorf("Result ID mismatch")
		}
	}

	// Test Search by Header
	results, err = svc.Search("ReqHeader.User-Agent:Go-Test-Client")
	if err != nil {
		t.Fatalf("Search Header failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for Header search, got %d", len(results))
	}

	// Test Search by Body
	results, err = svc.Search("ReqBody:value")
	if err != nil {
		t.Fatalf("Search Body failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for Body search, got %d", len(results))
	}
	
	// Test Search No Results
	results, err = svc.Search("NonExistent")
	if err != nil {
		t.Fatalf("Search NonExistent failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestFlowEntry_Conversion(t *testing.T) {
	flowID := uuid.NewV4()
	flow := &proxy.Flow{
		Id: flowID,
		ConnContext: &proxy.ConnContext{
			ClientConn: &proxy.ClientConn{},
		},
		Request: &proxy.Request{
			Method: "GET",
			URL:    &url.URL{Scheme: "http", Host: "test.com", Path: "/"},
			Proto:  "HTTP/1.1",
			Header: http.Header{"X-Test": []string{"true"}},
		},
		Response: &proxy.Response{
			StatusCode: 200,
			Header:     http.Header{"X-Response": []string{"ok"}},
		},
	}
	
	// Test NewFlowEntry
	entry, err := NewFlowEntry(flow)
	if err != nil {
		t.Fatalf("NewFlowEntry failed: %v", err)
	}
	
	if entry.ID != flowID.String() {
		t.Errorf("ID mismatch")
	}
	if entry.Method != "GET" {
		t.Errorf("Method mismatch")
	}
	
	// Test ToProxyFlow
	pFlow, err := entry.ToProxyFlow()
	if err != nil {
		t.Fatalf("ToProxyFlow failed: %v", err)
	}
	if pFlow.Id != flowID {
		t.Errorf("Restored ID mismatch")
	}
	
	// Check headers restored
	// Note: ToProxyFlow currently reconstructs headers but doesn't attach them to a fully functional Request object in the current partial implementation
	// We just check if it runs without error for now as per current implementation
}

func TestService_Save_Error(t *testing.T) {
	// Test handling of invalid flow (e.g. nil response if logic allows, or just verify Save handles errors)
	tmpDir := t.TempDir()
	svc, _ := NewService(tmpDir)
	defer svc.Close()
	
	// Close DB to force error on Save
	svc.db.Close()
	
	flow := &proxy.Flow{
		Id: uuid.NewV4(),
		ConnContext: &proxy.ConnContext{ClientConn: &proxy.ClientConn{}},
		Request: &proxy.Request{URL: &url.URL{}},
	}
	
	if err := svc.Save(flow); err == nil {
		t.Error("Expected error when saving to closed DB, got nil")
	}
}
