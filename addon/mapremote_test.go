package addon

import (
	"net/url"
	"os"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestMapRemote(t *testing.T) {
	mr := &MapRemote{
		Enable: true,
		Items: []*mapRemoteItem{
			{
				Enable: true,
				From: &MapFrom{
					Host: "example.com",
					Path: "/old",
				},
				To: &mapRemoteTo{
					Host: "new.example.com",
					Path: "/new",
				},
			},
			{
				Enable: true,
				From: &MapFrom{
					Host: "wildcard.com",
					Path: "/sub/*",
				},
				To: &mapRemoteTo{
					Host: "wildcard.new.com",
					Path: "/newsub",
				},
			},
		},
	}

	// Case 1: Simple replace
	req := &proxy.Request{Method: "GET", URL: &url.URL{Scheme: "http", Host: "example.com", Path: "/old"}}
	f := proxy.NewFlow()
	f.Request = req
	
	mr.Requestheaders(f)
	
	if f.Request.URL.Host != "new.example.com" {
		t.Errorf("Host mismatch: got %v, want new.example.com", f.Request.URL.Host)
	}
	if f.Request.URL.Path != "/new" {
		t.Errorf("Path mismatch: got %v, want /new", f.Request.URL.Path)
	}
	
	// Case 2: Wildcard
	req2 := &proxy.Request{Method: "GET", URL: &url.URL{Scheme: "http", Host: "wildcard.com", Path: "/sub/foo/bar"}}
	f2 := proxy.NewFlow()
	f2.Request = req2
	
	mr.Requestheaders(f2)
	
	if f2.Request.URL.Host != "wildcard.new.com" {
		t.Errorf("Host mismatch: got %v, want wildcard.new.com", f2.Request.URL.Host)
	}
	if f2.Request.URL.Path != "/newsub/foo/bar" {
		t.Errorf("Path mismatch: got %v, want /newsub/foo/bar", f2.Request.URL.Path)
	}
	
	if err := mr.validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

    // Case 3: Protocol change
    mr.Items[0].To.Protocol = "https"
    f3 := proxy.NewFlow()
    f3.Request = &proxy.Request{Method: "GET", URL: &url.URL{Scheme: "http", Host: "example.com", Path: "/old"}}
    mr.Requestheaders(f3)
    if f3.Request.URL.Scheme != "https" {
        t.Errorf("Scheme mismatch: got %v, want https", f3.Request.URL.Scheme)
    }

    // Case 4: Disabled addon
    mr.Enable = false
    f4 := proxy.NewFlow()
    f4.Request = &proxy.Request{Method: "GET", URL: &url.URL{Scheme: "http", Host: "example.com", Path: "/old"}}
    mr.Requestheaders(f4)
    if f4.Request.URL.Host == "new.example.com" {
        t.Error("Expected no change for disabled addon")
    }
}

func TestMapRemote_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mr      *MapRemote
		wantErr bool
	}{
		{
			name: "Valid",
			mr: &MapRemote{Items: []*mapRemoteItem{{From: &MapFrom{}, To: &mapRemoteTo{Host: "h"}}}},
			wantErr: false,
		},
		{
			name: "Missing From",
			mr: &MapRemote{Items: []*mapRemoteItem{{To: &mapRemoteTo{Host: "h"}}}},
			wantErr: true,
		},
		{
			name: "Missing To",
			mr: &MapRemote{Items: []*mapRemoteItem{{From: &MapFrom{}}}},
			wantErr: true,
		},
		{
			name: "Invalid Protocol",
			mr: &MapRemote{Items: []*mapRemoteItem{{From: &MapFrom{}, To: &mapRemoteTo{Protocol: "ftp"}}}},
			wantErr: true,
		},
        {
			name: "Empty To",
			mr: &MapRemote{Items: []*mapRemoteItem{{From: &MapFrom{}, To: &mapRemoteTo{}}}},
			wantErr: true,
		},
		{
			name: "Invalid Protocol",
			mr: &MapRemote{Items: []*mapRemoteItem{{From: &MapFrom{}, To: &mapRemoteTo{Protocol: "ftp"}}}},
			wantErr: true,
		},
		{
			name: "Missing From",
			mr: &MapRemote{Items: []*mapRemoteItem{{To: &mapRemoteTo{Path: "p"}}}},
			wantErr: true,
		},
		{
			name: "Missing To",
			mr: &MapRemote{Items: []*mapRemoteItem{{From: &MapFrom{}}}},
			wantErr: true,
		},
		{
			name: "Empty To",
			mr: &MapRemote{Items: []*mapRemoteItem{{From: &MapFrom{}, To: &mapRemoteTo{}}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.mr.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMapRemote_ItemMatchDisabled(t *testing.T) {
    item := &mapRemoteItem{Enable: false, From: &MapFrom{Host: "h"}}
    if item.match(&proxy.Request{URL: &url.URL{Host: "h"}}) {
        t.Error("Disabled item should not match")
    }
}

func TestMapRemote_NewFromFile(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "mapremote.json")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(`{"enable": true, "items": []}`)
	tmpFile.Close()

	mr, err := NewMapRemoteFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewMapRemoteFromFile failed: %v", err)
	}
	if !mr.Enable {
		t.Error("Expected enabled from file")
	}
}
