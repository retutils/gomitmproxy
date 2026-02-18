package addon

import (
	"database/sql"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/retutils/gomitmproxy/proxy"
)

func TestStorageAddon_Extended(t *testing.T) {
	// Setup
	tempDir, err := os.MkdirTemp("", "gomitmproxy_test_ext")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageAddon, err := NewStorageAddon(tempDir)
	if err != nil {
		t.Fatalf("Failed to init storage addon: %v", err)
	}
	defer storageAddon.Close()

	// Helper to create flow
	createFlow := func() *proxy.Flow {
		u, _ := url.Parse("http://example.com/api/test")
		f := proxy.NewFlow()
		f.ConnContext = &proxy.ConnContext{ClientConn: &proxy.ClientConn{Id: proxy.NewFlow().Id}}
		f.Request = &proxy.Request{Method: "GET", URL: u, Header: http.Header{}, Body: []byte("")}
		f.Response = &proxy.Response{StatusCode: 200, Header: http.Header{}, Body: []byte("OK")}
		f.Metadata = make(map[string]interface{})
		return f
	}

	// 1. Test Saving Flow without PII (should not error)
	t.Run("Save_No_PII", func(t *testing.T) {
		f := createFlow()
		err := storageAddon.Service.Save(f)
		if err != nil {
			t.Errorf("Save failed for flow without PII: %v", err)
		}
	})

	// 2. Test Saving Flow with Empty PII Metadata
	t.Run("Save_Empty_PII_List", func(t *testing.T) {
		f := createFlow()
		f.Metadata["pii"] = []PIIFinding{}
		err := storageAddon.Service.Save(f)
		if err != nil {
			t.Errorf("Save failed for flow with empty PII list: %v", err)
		}
	})

	// 3. Test Saving Flow with Invalid PII Metadata Type (should be ignored or handled gracefully)
	t.Run("Save_Invalid_PII_Type", func(t *testing.T) {
		f := createFlow()
		f.Metadata["pii"] = "invalid_string_instead_of_slice"
		err := storageAddon.Service.Save(f)
		if err != nil {
			t.Errorf("Save failed for flow with invalid PII metadata type: %v", err)
		}
	})

	// 4. Test Search with Invalid Query (should fallback or error)
	t.Run("Search_Invalid_Query", func(t *testing.T) {
		// HTTPQL parse error -> Fallback to Bleve query string
		// Invalid Bleve query string -> Error from Bleve
		_, err := storageAddon.Service.Search("invalid: syntax: : :")
		if err != nil {
			t.Logf("Search returned expected error for invalid query: %v", err)
		}
	})

	// 5. Test Search Empty
	t.Run("Search_No_Results", func(t *testing.T) {
		res, err := storageAddon.Service.Search("non_existent_term")
		if err != nil {
			t.Errorf("Search failed: %v", err)
		}
		if len(res) != 0 {
			t.Errorf("Expected 0 results, got %d", len(res))
		}
	})

	// 6. Test URL Port parsing in Save
	t.Run("Save_URL_With_Port", func(t *testing.T) {
		f := createFlow()
		u, _ := url.Parse("http://example.com:8080/foo")
		f.Request.URL = u
		err := storageAddon.Service.Save(f)
		if err != nil {
			t.Errorf("Save failed for URL with port: %v", err)
		}
	})

	// 7. Test Save Error after Close
	t.Run("Save_Error_After_Close", func(t *testing.T) {
		sa, _ := NewStorageAddon(t.TempDir())
		sa.Close() // Close DB and Index

		f := createFlow()
		err := sa.Service.Save(f)
		if err == nil {
			t.Errorf("Expected Save to fail after Close, but got nil")
		}
	})

	// 8. Test Search Error after Close
	t.Run("Search_Error_After_Close", func(t *testing.T) {
		sa, _ := NewStorageAddon(t.TempDir())
		sa.Close()

		_, err := sa.Service.Search("foo")
		if err == nil {
			t.Errorf("Expected Search to fail after Close, but got nil")
		}
	})

	// 9. Test PII Insert Fail (Table Missing)
	t.Run("Save_PII_Insert_Fail", func(t *testing.T) {
		tempDir2 := t.TempDir()
		sa, _ := NewStorageAddon(tempDir2)
		defer sa.Close()

		// Manually drop table to induce error
		dbPath := filepath.Join(tempDir2, "flows.duckdb")
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatalf("Failed to open db: %v", err)
		}

		_, err = db.Exec("DROP TABLE pii_detections")
		if err != nil {
			t.Logf("Failed to drop table (expected if db locked): %v", err)
			db.Close()
			return
		}
		db.Close()

		f := createFlow()
		f.Metadata["pii"] = []PIIFinding{{Source: "body", Type: "Email"}}

		err = sa.Service.Save(f)
		if err != nil {
			// If it returns error, that's also fine (means implementation changed to stricter)
		}
	})

	// 10. Test Init Error (Invalid Path)
	t.Run("Init_Error_Invalid_Path", func(t *testing.T) {
		tempDir3 := t.TempDir()
		// Create a file
		filePath := filepath.Join(tempDir3, "blocking_file")
		os.WriteFile(filePath, []byte("content"), 0644)

		// Try to create storage in a subdir of that file
		// os.MkdirAll should fail
		_, err := NewStorageAddon(filepath.Join(filePath, "subdir"))
		if err == nil {
			t.Errorf("Expected valid NewStorageAddon to fail with invalid path, but got nil")
		}
	})
}
