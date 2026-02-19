package storage

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
		Metadata: make(map[string]interface{}),
	}

	// Test Save
	entry, _ := NewFlowEntry(flow)
	if err := svc.SaveEntry(entry, flow.Metadata["pii"]); err != nil {
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

	// Test Save with PII
	flow.Id = uuid.NewV4()
	flow.Metadata["pii"] = []map[string]string{
		{"source": "body", "type": "Email", "snippet": "test@example.com"},
	}
	entry, _ = NewFlowEntry(flow)
	if err := svc.SaveEntry(entry, flow.Metadata["pii"]); err != nil {
		t.Fatalf("SaveEntry with PII failed: %v", err)
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

func TestFlowEntry_ToProxyFlow_Error(t *testing.T) {
    entry := &FlowEntry{
        ID: "invalid-uuid",
        RequestHeader: "{}",
        ResponseHeader: "{}",
    }
    _, err := entry.ToProxyFlow()
    if err == nil {
        t.Error("Expected error for invalid UUID")
    }

    entry = &FlowEntry{
        ID: uuid.NewV4().String(),
        RequestHeader: "{invalid-json}",
        ResponseHeader: "{}",
    }
    _, err = entry.ToProxyFlow()
    if err == nil {
        t.Error("Expected error for invalid RequestHeader JSON")
    }

    entry = &FlowEntry{
        ID: uuid.NewV4().String(),
        RequestHeader: "{}",
        ResponseHeader: "{invalid-json}",
    }
    _, err = entry.ToProxyFlow()
    if err == nil {
        t.Error("Expected error for invalid ResponseHeader JSON")
    }
}

func TestFlowEntry_NewFlowEntry_PIIBool(t *testing.T) {
    f := &proxy.Flow{
        Id: uuid.NewV4(),
        ConnContext: &proxy.ConnContext{ClientConn: &proxy.ClientConn{}},
        Request: &proxy.Request{URL: &url.URL{}},
        Metadata: map[string]interface{}{"pii": true},
    }
    entry, err := NewFlowEntry(f)
    if err != nil { t.Fatal(err) }
    if !entry.HasPII {
        t.Error("Expected HasPII true")
    }
}

func TestService_Save_Error(t *testing.T) {
	// Test handling of invalid flow (e.g. nil response if logic allows, or just verify Save handles errors)
	tmpDir := t.TempDir()
	svc, _ := NewService(tmpDir)
	defer svc.Close()
	
    flow := &proxy.Flow{
		Id: uuid.NewV4(),
		ConnContext: &proxy.ConnContext{ClientConn: &proxy.ClientConn{}},
		Request: &proxy.Request{URL: &url.URL{}},
	}

	// 1. Database Closed Error
	svc.db.Close()
	entry, _ := NewFlowEntry(flow)
	if err := svc.SaveEntry(entry, flow.Metadata["pii"]); err == nil {
		t.Error("Expected error when saving to closed DB, got nil")
	}

    // 2. Bleve Closed Error
    svc2, _ := NewService(t.TempDir())
    svc2.index.Close()
    entry2, _ := NewFlowEntry(flow)
    if err := svc2.SaveEntry(entry2, nil); err == nil {
        t.Error("Expected error when indexing to closed index")
    }
}

func TestService_Search_MissingFromDB(t *testing.T) {
    tmpDir := t.TempDir()
    svc, _ := NewService(tmpDir)
    defer svc.Close()

    // Save normally
    flow := &proxy.Flow{Id: uuid.NewV4(), Request: &proxy.Request{URL: &url.URL{Host: "missing.com"}}, ConnContext: &proxy.ConnContext{ClientConn: &proxy.ClientConn{}}}
    entry, _ := NewFlowEntry(flow)
    svc.SaveEntry(entry, nil)

    // Manually delete from DB but KEEP in index
    svc.db.Exec("DELETE FROM flows WHERE id = ?", entry.ID)

    // Search should skip the missing DB entry
    results, err := svc.Search("missing.com")
    if err != nil { t.Fatal(err) }
    if len(results) != 0 {
        t.Errorf("Expected 0 results for missing DB entry, got %d", len(results))
    }
}

func TestService_Close_Nil(t *testing.T) {
    s := &Service{}
    if err := s.Close(); err != nil {
        t.Errorf("Expected nil error for empty service close, got %v", err)
    }
}

func TestService_Search_Error(t *testing.T) {
    tmpDir := t.TempDir()
    svc, _ := NewService(tmpDir)
    // defer svc.Close() - don't defer because we close manually to trigger error

    // Search on closed index
    svc.Close()
    _, err := svc.Search("foo")
    if err == nil {
        t.Error("Expected error searching closed index")
    }
}

func TestNewService_Error(t *testing.T) {
    // 1. Invalid path (file instead of dir)
    tmpFile, _ := os.CreateTemp("", "blocked")
    defer os.Remove(tmpFile.Name())
    tmpFile.Close()
    
    _, err := NewService(filepath.Join(tmpFile.Name(), "subdir"))
    if err == nil {
        t.Error("Expected error for invalid storage path")
    }

    // 2. Schema init error (hard to trigger without mocking DB)
}
