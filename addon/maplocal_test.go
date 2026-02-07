package addon

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestMapLocal(t *testing.T) {
	// Setup temp dir and files
	tmpDir := t.TempDir()
	fooFile := filepath.Join(tmpDir, "foo.txt")
	os.WriteFile(fooFile, []byte("foo content"), 0644)
	
	barDir := filepath.Join(tmpDir, "bar")
	os.Mkdir(barDir, 0755)
	barFile := filepath.Join(barDir, "baz.txt")
	os.WriteFile(barFile, []byte("bar baz content"), 0644)

	// Test 1: File to File
	ml := &MapLocal{
		Enable: true,
		Items: []*mapLocalItem{
			{
				Enable: true,
				From: &MapFrom{
					Host: "example.com",
					Path: "/foo",
				},
				To: &mapLocalTo{
					Path: fooFile,
				},
			},
            {
				Enable: true,
				From: &MapFrom{
					Host: "example.com",
					Path: "/bar/*", // Wildcard
				},
				To: &mapLocalTo{
					Path: barDir,
				},
			},
		},
	}

	// Case 1: Match file
	req := &proxy.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "http", Host: "example.com", Path: "/foo"},
	}
	f := proxy.NewFlow()
	f.Request = req
	
	ml.Requestheaders(f)
	
	if f.Response == nil {
		t.Fatal("Expected response, got nil")
	}
	if f.Response.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", f.Response.StatusCode)
	}
    // Check body if possible, BodyReader is set
    // In real flow, BodyReader is read. Here we can check it.
    // But BodyReader is io.Reader.

	// Case 2: Match dir wildcard
    req2 := &proxy.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "http", Host: "example.com", Path: "/bar/baz.txt"},
	}
    f2 := proxy.NewFlow()
	f2.Request = req2
    
    ml.Requestheaders(f2)
    
    if f2.Response == nil {
		t.Fatal("Expected response for dir match, got nil")
	}
    if f2.Response.StatusCode != 200 {
		t.Errorf("Expected 200 for dir match, got %d", f2.Response.StatusCode)
	}
    
    // Case 3: No match
    req3 := &proxy.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "http", Host: "example.com", Path: "/nomatch"},
	}
    f3 := proxy.NewFlow()
    f3.Request = req3
    
    ml.Requestheaders(f3)
    if f3.Response != nil {
        t.Error("Expected no response for no match")
    }
    
    // Test validation
    if err := ml.validate(); err != nil {
        t.Errorf("Validation failed: %v", err)
    }
}
