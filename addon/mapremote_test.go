package addon

import (
	"net/url"
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
	
	// Validation
	if err := mr.validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}
}
