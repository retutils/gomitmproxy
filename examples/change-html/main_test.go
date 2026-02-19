package main

import (
	"net/http"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestChangeHtml(t *testing.T) {
	addon := &ChangeHtml{}
	f := &proxy.Flow{
		Response: &proxy.Response{
			Header: http.Header{"Content-Type": []string{"text/html"}},
			Body:   []byte("<title>test</title>"),
		},
	}
	addon.Response(f)
	if string(f.Response.Body) != "<title>test - go-mitmproxy</title>" {
		t.Errorf("Unexpected body: %s", string(f.Response.Body))
	}

    // Skip non-html
    f2 := &proxy.Flow{
		Response: &proxy.Response{
			Header: http.Header{"Content-Type": []string{"text/plain"}},
			Body:   []byte("<title>test</title>"),
		},
	}
    addon.Response(f2)
    if string(f2.Response.Body) != "<title>test</title>" {
        t.Error("Should not change non-html")
    }
}
