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

func createTestFlow() *proxy.Flow {
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
    req, _ := http.NewRequest("POST", "http://example.com/foo", bytes.NewReader(f.Request.Body))
    f.Request.SetRaw(req)
	f.Request.Header.Set("Content-Type", "text/plain")
	f.Response.Header.Set("Content-Type", "text/plain")
    return f
}

func TestDumper(t *testing.T) {
	var buf bytes.Buffer
	dumper := NewDumper(&buf, 1)

	t.Run("Standard", func(t *testing.T) {
		buf.Reset()
		f := createTestFlow()
		go func() { f.Finish() }()
		dumper.Requestheaders(f)
		time.Sleep(100 * time.Millisecond)

		output := buf.String()
		if !contains(output, "POST /foo HTTP/1.1") || !contains(output, "request body") || !contains(output, "response body") {
			t.Errorf("Standard dump output incorrect: %s", output)
		}
	})

	t.Run("Binary", func(t *testing.T) {
		buf.Reset()
		f := createTestFlow()
		f.Response.Header.Set("Content-Type", "image/png")
		f.Response.Body = []byte{0x89, 'P', 'N', 'G'}
		go func() { f.Finish() }()
		dumper.Requestheaders(f)
		time.Sleep(100 * time.Millisecond)
		if contains(buf.String(), "PNG") {
			t.Error("Binary body should be filtered")
		}
	})

	t.Run("Level0", func(t *testing.T) {
		buf.Reset()
		dumper0 := NewDumper(&buf, 0)
		f := createTestFlow()
		go func() { f.Finish() }()
		dumper0.Requestheaders(f)
		time.Sleep(100 * time.Millisecond)
		if contains(buf.String(), "request body") {
			t.Error("Level 0 should not include body")
		}
	})
}

func TestDumper_NewWithFilename(t *testing.T) {
	tmpFile := t.TempDir() + "/dump.log"
	d := NewDumperWithFilename(tmpFile, 1)
	if d == nil {
		t.Fatal("Expected non-nil dumper")
	}
}

func TestDumper_EdgeCases(t *testing.T) {
	var buf bytes.Buffer
	
	t.Run("InvalidLevel", func(t *testing.T) {
		d := NewDumper(&buf, 99)
		if d.level != 0 {
			t.Errorf("Expected level 0 for invalid input, got %d", d.level)
		}
	})

	t.Run("RequestExtras", func(t *testing.T) {
		buf.Reset()
		dumper := NewDumper(&buf, 1)
		f := createTestFlow()
		f.Request.Raw().TransferEncoding = []string{"chunked"}
		f.Request.Raw().Close = true
		
		dumper.dump(f) // Call dump directly to avoid sleep
		output := buf.String()
		if !contains(output, "Transfer-Encoding: chunked") {
			t.Error("Missing Transfer-Encoding")
		}
		if !contains(output, "Connection: close") {
			t.Error("Missing Connection: close")
		}
	})

	t.Run("NonPrintableRequest", func(t *testing.T) {
		buf.Reset()
		dumper := NewDumper(&buf, 1)
		f := createTestFlow()
		f.Request.Body = []byte{0x00, 0x01, 0x02} // Non-printable
		
		dumper.dump(f)
		if contains(buf.String(), "\x00") {
			t.Error("Non-printable request body should be filtered")
		}
	})
}

func contains(s, substr string) bool {
    return bytes.Contains([]byte(s), []byte(substr))
}
