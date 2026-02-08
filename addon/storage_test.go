package addon

import (
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	
	"github.com/retutils/gomitmproxy/proxy"
	uuid "github.com/satori/go.uuid"
)

func TestStorageAddon_SaveAndSearch(t *testing.T) {
	// Create temp dir for storage
	tmpDir := t.TempDir()
	storageDir := filepath.Join(tmpDir, "mitm_storage_test")

	// Initialize StorageAddon
	addon, err := NewStorageAddon(storageDir)
	if err != nil {
		t.Fatalf("Failed to create storage addon: %v", err)
	}
	defer addon.Close()

	// Create a dummy flow
	flow := &proxy.Flow{
		Id: uuid.NewV4(),
		ConnContext: &proxy.ConnContext{
			ClientConn: &proxy.ClientConn{},
		},
		Request: &proxy.Request{
			Method: "POST",
			URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/login"},
			Proto:  "HTTP/1.1",
			Header: http.Header{"Content-Type": []string{"application/json"}},
			Body:   []byte(`{"username": "testuser", "password": "password123"}`),
		},
		Response: &proxy.Response{
			StatusCode: 200,
			Header:     http.Header{"Server": []string{"nginx"}},
			Body:       []byte(`{"token": "abcdef123456"}`),
		},
	}
	// Mock URL string wrapper if needed, but gomitmproxy URL is *url.URL
	
	// Save flow
	err = addon.Service.Save(flow)
	if err != nil {
		t.Fatalf("Failed to save flow: %v", err)
	}

	// Allow some time for indexing (Bleve is usually fast but async in some configs, here it's sync in Save?)
	// Our Save implementation calls index.Index which is synchronous for the index writer but might have slight delay for visibility? 
	// Usually bleve.Index is blocking.
	
	// Search by URL
	results, err := addon.Service.Search("example.com")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least 1 result, got 0")
	}

	found := results[0]
	if found.URL != "https://example.com/api/login" {
		t.Errorf("Expected URL 'https://example.com/api/login', got '%s'", found.URL)
	}
	if string(found.RequestBody) != `{"username": "testuser", "password": "password123"}` {
		t.Errorf("Request body mismatch")
	}

	// Search by Body content
	results, err = addon.Service.Search("testuser")
	if err != nil {
		t.Fatalf("Search by body failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected finding 'testuser' in body, got 0 results")
	}

	// Search by Header Field
	// Bleve query syntax for field: fieldname:value
	// Since we mapped ReqHeader as a dynamic object, we should be able to search ReqHeader.Content-Type
	results, err = addon.Service.Search("ReqHeader.Content-Type:application/json")
	if err != nil {
		t.Fatalf("Search by header failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected finding 'Content-Type:application/json' in header, got 0 results")
	}
}
