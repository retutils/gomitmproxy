package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestChangeHtml(t *testing.T) {
	addon := &ChangeHtml{}
	
	f := &proxy.Flow{
		Response: &proxy.Response{
			Header: http.Header{"Content-Type": []string{"text/html"}},
			Body:   []byte("<html><head><title>Original</title></head></html>"),
		},
	}
	
	addon.Response(f)
	if !strings.Contains(string(f.Response.Body), "Original - go-mitmproxy") {
		t.Error("Title not modified correctly")
	}
	
	// Test non-html
	f2 := &proxy.Flow{
		Response: &proxy.Response{
			Header: http.Header{"Content-Type": []string{"text/plain"}},
			Body:   []byte("Original"),
		},
	}
	addon.Response(f2)
	if string(f2.Response.Body) != "Original" {
		t.Error("Non-HTML should not be modified")
	}
}

func TestRun(t *testing.T) {
	// Success case
	go Run(":0")
	time.Sleep(100 * time.Millisecond)

	// Error case (invalid address)
	err := Run("invalid-addr:999999")
	if err == nil {
		t.Error("Expected error for invalid address")
	}
}
