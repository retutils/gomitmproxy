package addon

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestStorageAddon_Integration_FullFlow(t *testing.T) {
	// 1. Setup Storage Path
	tmpDir := t.TempDir()
	storageDir := filepath.Join(tmpDir, "mitm_integration_storage")
	
	// 2. Setup Target Server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Custom-Header", "Found-It")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Received: %s", body)))
	}))
	defer targetServer.Close()
	
	// 3. Setup Proxy
	opts := &proxy.Options{
		Addr:              ":0", // Random port
		StreamLargeBodies: 1024 * 1024 * 5,
	}
	p, err := proxy.NewProxy(opts)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}
	
	storageAddon, err := NewStorageAddon(storageDir)
	if err != nil {
		t.Fatalf("Failed to create storage addon: %v", err)
	}
	defer storageAddon.Close()
	p.AddAddon(storageAddon)
	
	go p.Start()
	
	// Wait for proxy to bind port
	var proxyAddr string
	for i := 0; i < 20; i++ {
		proxyAddr = p.Addr()
		if proxyAddr != ":0" && proxyAddr != "" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if proxyAddr == ":0" || proxyAddr == "" {
		t.Fatal("Proxy failed to start or update address")
	}
	t.Logf("Proxy running at %s", proxyAddr)

	// 4. Create Client using Proxy
	proxyURL, err := url.Parse("http://" + proxyAddr)
	if err != nil {
		t.Fatalf("Invalid proxy URL: %v", err)
	}
	
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}
	
	// 5. Send Request
	reqBody := "search-query-123"
	req, err := http.NewRequest("POST", targetServer.URL, strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	bodyBytes, _ := io.ReadAll(resp.Body)
	respBody := string(bodyBytes)
	
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(respBody, "Received: search-query-123") {
		t.Errorf("Unexpected response body: %s", respBody)
	}
	
	// 6. Verify Storage
	// Wait for async save
	time.Sleep(1 * time.Second)
	
	// Search by contents
	// Using URL might fail due to special characters in query string syntax if not escaped.
	// Let's search by unique body content "search-query-123"
	results, err := storageAddon.Service.Search("search-query-123")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected to find request in storage, got 0 results")
	}

	// Verify details
	found := results[0]
	if string(found.RequestBody) != reqBody {
		t.Errorf("Stored request body mismatch: expected %s, got %s", reqBody, found.RequestBody)
	}
	if found.StatusCode != 200 {
		t.Errorf("Stored status code mismatch: expected 200, got %d", found.StatusCode)
	}
	if found.Method != "POST" {
		t.Errorf("Stored method mismatch: expected POST, got %s", found.Method)
	}
	
	// Search by Response Header
	// Since mapping is dynamic for headers, value should be searchable
	results, err = storageAddon.Service.Search("Found-It")
	if err != nil {
		t.Fatalf("Search header failed: %v", err)
	}
	if len(results) == 0 {
		t.Errorf("Expected to find request by response header 'Found-It', got 0 results")
	}

	// 7. Test HTTPQL
	// req.method.eq:"POST" AND resp.code.eq:200
	httpqlQuery := `req.method.eq:"POST" AND resp.code.eq:200`
	results, err = storageAddon.Service.Search(httpqlQuery)
	if err != nil {
		t.Fatalf("HTTPQL Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Errorf("Expected to find request by HTTPQL '%s', got 0 results", httpqlQuery)
	}

	// Negative Test for HTTPQL
	httpqlQueryNegative := `req.method.eq:"GET"`
	results, err = storageAddon.Service.Search(httpqlQueryNegative)
	if err != nil {
		t.Fatalf("HTTPQL Negative Search failed: %v", err)
	}
	if len(results) > 0 {
		t.Errorf("Expected 0 results for HTTPQL '%s', got %d", httpqlQueryNegative, len(results))
	}

	// 8. Test HTTPQL Body Search
	// Body was "search-query-123", search for phrase "search-query"
	httpqlBodyQuery := `req.body.cont:"search-query"`
	results, err = storageAddon.Service.Search(httpqlBodyQuery)
	if err != nil {
		t.Fatalf("HTTPQL Body Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Errorf("Expected to find request by HTTPQL Body '%s', got 0 results", httpqlBodyQuery)
	}
}
