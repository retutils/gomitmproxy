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
    
    if err := ml.validate(); err != nil {
        t.Errorf("Validation failed: %v", err)
    }

    // Case 4: Disabled addon
    ml.Enable = false
    f4 := proxy.NewFlow()
    f4.Request = req
    ml.Requestheaders(f4)
    if f4.Response != nil {
        t.Error("Expected no response for disabled addon")
    }
    ml.Enable = true

    // Case 5: Item disabled
    ml.Items[0].Enable = false
    f5 := proxy.NewFlow()
    f5.Request = req
    ml.Requestheaders(f5)
    if f5.Response != nil {
        t.Error("Expected no response for disabled item")
    }
    ml.Items[0].Enable = true

    // Case 6: Map to non-existent file
    ml.Items[0].To.Path = "/non/existent/file"
    f6 := proxy.NewFlow()
    f6.Request = req
    ml.Requestheaders(f6)
    if f6.Response == nil {
        t.Fatal("Expected response even if file missing (it should set 404 or similar)")
    }
    if f6.Response.StatusCode != 404 {
        t.Errorf("Expected 404 for missing file, got %d", f6.Response.StatusCode)
    }

    // Case 7: Match directory without wildcard
    ml.Items[1].From.Path = "/bar"
    f7 := proxy.NewFlow()
    f7.Request = &proxy.Request{Method: "GET", URL: &url.URL{Host: "example.com", Path: "/bar/baz.txt"}}
    ml.Requestheaders(f7)
    if f7.Response != nil {
        t.Error("Expected no match for directory without wildcard")
    }
}

func TestMapLocal_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ml      *MapLocal
		wantErr bool
	}{
		{
			name: "Valid",
			ml: &MapLocal{Items: []*mapLocalItem{{From: &MapFrom{}, To: &mapLocalTo{Path: "p"}}}},
			wantErr: false,
		},
		{
			name: "Missing From",
			ml: &MapLocal{Items: []*mapLocalItem{{To: &mapLocalTo{Path: "p"}}}},
			wantErr: true,
		},
		{
			name: "Missing To",
			ml: &MapLocal{Items: []*mapLocalItem{{From: &MapFrom{}}}},
			wantErr: true,
		},
        {
			name: "Empty To Path",
			ml: &MapLocal{Items: []*mapLocalItem{{From: &MapFrom{}, To: &mapLocalTo{Path: ""}}}},
			wantErr: true,
		},
        {
            name: "Nil From",
            ml: &MapLocal{Items: []*mapLocalItem{{To: &mapLocalTo{Path: "p"}}}},
            wantErr: true,
        },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ml.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMapLocal_NewFromFile(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "maplocal.json")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(`{"enable": true, "items": []}`)
	tmpFile.Close()

	ml, err := NewMapLocalFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewMapLocalFromFile failed: %v", err)
	}
	if !ml.Enable {
		t.Error("Expected enabled from file")
	}
}

func TestMapLocal_DirMapping_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := tmpDir + "/subdir"
	os.Mkdir(subDir, 0755)
	
	item := &mapLocalItem{
		Enable: true,
		From:   &MapFrom{Path: "/static/*"},
		To:     &mapLocalTo{Path: subDir},
	}
	
	// 1. Request matches dir exactly (should fail if it's a directory)
	req := &proxy.Request{URL: &url.URL{Path: "/static/"}}
	_, resp := item.response(req)
	if resp.StatusCode != 500 {
		t.Errorf("Expected 500 for mapping to directory, got %d", resp.StatusCode)
	}
	
	// 2. Request matches file in dir
	os.WriteFile(subDir+"/test.txt", []byte("ok"), 0644)
	req2 := &proxy.Request{URL: &url.URL{Path: "/static/test.txt"}}
	_, resp2 := item.response(req2)
	if resp2.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp2.StatusCode)
	}
}
