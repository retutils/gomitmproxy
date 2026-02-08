package proxy

import (
	"net/http"
	"net/url"
	"testing"
    
    uuid "github.com/satori/go.uuid"
)

func BenchmarkFlow_MarshalJSON(b *testing.B) {
    f := NewFlow()
    f.Id = uuid.NewV4()
    f.Request = NewRequest(&http.Request{
        Method: "GET",
        URL: &url.URL{
            Scheme: "http",
            Host:   "example.com",
            Path:   "/foo/bar/baz?query=123",
        },
        Header: make(http.Header),
    })
    f.Request.Header.Set("User-Agent", "Benchmark/1.0")
    f.Request.Header.Set("Content-Type", "application/json")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := f.MarshalJSON()
        if err != nil {
            b.Fatal(err)
        }
    }
}


