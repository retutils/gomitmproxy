package addon

import (
	"net/url"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func BenchmarkMapFrom_Match_Simple(b *testing.B) {
    m := &MapFrom{
        Host: "example.com",
        Method: []string{"GET"},
        Path: "/foo",
    }
    
    req := &proxy.Request{
        Method: "GET",
        URL: &url.URL{
            Scheme: "http",
            Host:   "example.com",
            Path:   "/foo/bar",
        },
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        m.Match(req)
    }
}

func BenchmarkMapFrom_Match_Wildcard(b *testing.B) {
    m := &MapFrom{
        Host: "example.com",
        Path: "/foo/*/bar", // Wildcard matching
    }
    
    req := &proxy.Request{
        Method: "GET",
        URL: &url.URL{
            Scheme: "http",
            Host:   "example.com",
            Path:   "/foo/123/bar",
        },
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        m.Match(req)
    }
}
