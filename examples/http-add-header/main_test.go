package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestAddHeader(t *testing.T) {
	addon := &AddHeader{}
	
	f := &proxy.Flow{
		Response: &proxy.Response{
			Header: make(http.Header),
		},
	}
	
	addon.Responseheaders(f)
	if f.Response.Header.Get("x-count") != "1" {
		t.Errorf("Expected x-count 1, got %s", f.Response.Header.Get("x-count"))
	}
	
	addon.Responseheaders(f)
	vals := f.Response.Header.Values("x-count")
	found := false
	for _, v := range vals {
		if v == "2" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected x-count 2 to be in %v", vals)
	}
}

func TestRun(t *testing.T) {
	go Run()
	time.Sleep(100 * time.Millisecond)
}
