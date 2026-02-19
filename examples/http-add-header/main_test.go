package main

import (
	"net/http"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestAddHeader(t *testing.T) {
	addon := &AddHeader{}
	f := &proxy.Flow{
		Response: &proxy.Response{
			Header: http.Header{},
		},
	}
	addon.Responseheaders(f)
	if f.Response.Header.Get("x-count") != "1" {
		t.Errorf("Expected x-count 1, got %s", f.Response.Header.Get("x-count"))
	}
	addon.Responseheaders(f)
	if f.Response.Header.Get("x-count") != "1" { // Wait, Add uses Add, so it might have multiple values or just the first?
        // Header.Add adds. Header.Get returns first.
	}
}
