package addon

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/retutils/gomitmproxy/proxy"
)

func TestStorageAddon_PII_Integration_SaveAndSearch(t *testing.T) {
	// 1. Setup temp storage dir
	tempDir, err := os.MkdirTemp("", "gomitmproxy_test_pii")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Initialize Storage Addon
	storageAddon, err := NewStorageAddon(tempDir)
	if err != nil {
		t.Fatalf("Failed to init storage addon: %v", err)
	}
	defer storageAddon.Close()

	// 3. Initialize PII Addon
	piiAddon := NewPIIAddonStub()

	// 4. Create a flow with PII
	u, _ := url.Parse("http://example.com/api/user")
	f := proxy.NewFlow()

	// Create proper ConnContext with ClientConn to avoid panic in flow entry creation
	f.ConnContext = &proxy.ConnContext{
		ClientConn: &proxy.ClientConn{
			Id: proxy.NewFlow().Id, // Just need a UUID, reusing NewFlow().Id is hacky but works as it returns UUID
		},
	}

	f.Request = &proxy.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{},
		Body:   []byte(""),
	}
	f.Response = &proxy.Response{
		StatusCode: 200,
		Header:     http.Header{},
		Body:       []byte(`{"email": "test@example.com", "secret": "s3cr3t_t0k3n"}`),
	}
	f.Response.Header.Set("Content-Type", "application/json")

	// Ensure Metadata is initialized (NewFlow does it, but double check)
	if f.Metadata == nil {
		f.Metadata = make(map[string]interface{})
	}

	// 5. Run PII scanner
	piiAddon.Response(f)

	// Verify PII detected in Metadata
	if _, ok := f.Metadata["pii"]; !ok {
		t.Fatalf("PII scanner failed to detect PII")
	}

	// 6. Save Flow
	ctx := context.Background()
	err = saveFlow(storageAddon, f)
	if err != nil {
		t.Fatalf("Failed to save flow: %v", err)
	}

	// 7. Verify Data in DuckDB
	dbPath := filepath.Join(tempDir, "flows.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("Failed to open duckdb for verification: %v", err)
	}
	defer db.Close()

	var hasPII bool
	err = db.QueryRowContext(ctx, "SELECT has_pii FROM flows WHERE id = ?", f.Id.String()).Scan(&hasPII)
	if err != nil {
		t.Fatalf("Failed to query has_pii from flows: %v", err)
	}

	if !hasPII {
		t.Errorf("Expected has_pii to be true, got false")
	} else {
		t.Logf("Found has_pii=true in DB")
	}

	// 8. Verify Bleve Index via Search
	// Give indexing a moment
	timeout := time.After(2 * time.Second)
	tick := time.Tick(100 * time.Millisecond)
	found := false

	// Try querying for URL keyword first to see if index is ready
	for {
		select {
		case <-timeout:
			t.Errorf("Timeout waiting for search index")
			found = true // Force break outer loop logic
			break
		case <-tick:
			results, err := storageAddon.Service.Search("user") // "user" in URL
			if err == nil {
				for _, res := range results {
					if res.ID == f.Id.String() {
						found = true
						break
					}
				}
			}
		}
		if found {
			break
		}
	}

	if !found {
		t.Errorf("Flow not found in Bleve search for 'user'")
	}
}
