package addon

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
	"go.uber.org/atomic"
)

func TestDumper(t *testing.T) {
	// Setup
	var buf bytes.Buffer
	dumper := NewDumper(&buf, 1)

	// Create dummy flow
	f := proxy.NewFlow()
	f.Request = &proxy.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "http", Host: "example.com", Path: "/foo"},
		Proto:  "HTTP/1.1",
		Header: make(http.Header),
		Body:   []byte("request body"),
	}
	f.Response = &proxy.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       []byte("response body"),
	}
	f.ConnContext = &proxy.ConnContext{
		FlowCount: *atomic.NewUint32(0),
	}
    // Need to mock Request.Raw() if possible or populate it
    req, _ := http.NewRequest("POST", "http://example.com/foo", bytes.NewReader(f.Request.Body))
    f.Request.SetRaw(req)

	f.Request.Header.Set("Content-Type", "text/plain")
	f.Response.Header.Set("Content-Type", "text/plain")
    
    // We need to signal done, because Requestheaders waits for it
    go func() {
        time.Sleep(10 * time.Millisecond)
        f.Finish()
    }()

	dumper.Requestheaders(f)
    
    // Wait for dump to complete (async)
    // Since dump happens after f.Done(), and we call f.Finish(), it should run.
    // But we need to wait for the goroutine in Requestheaders to finish.
    time.Sleep(100 * time.Millisecond)

	output := buf.String()
	if output == "" {
		t.Error("expected output from dumper, got empty")
	}
	if !contains(output, "POST /foo HTTP/1.1") {
		t.Errorf("output missing request line. Got: %s", output)
	}
	if !contains(output, "request body") {
		t.Errorf("output missing request body")
	}
	if !contains(output, "response body") {
		t.Errorf("output missing response body")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(s)] == s &&  bytes.Contains([]byte(s), []byte(substr))
    // simple strings.Contains
    return bytes.Contains([]byte(s), []byte(substr))
}
