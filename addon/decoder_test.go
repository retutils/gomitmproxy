package addon

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestDecoder(t *testing.T) {
	decoder := &Decoder{}

	// Compress data
	original := []byte("hello world")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write(original)
	gw.Close()

	// Create flow
	f := proxy.NewFlow()
	f.Response = &proxy.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       buf.Bytes(),
	}
	f.Response.Header.Set("Content-Encoding", "gzip")

	// Verify before
	if f.Response.Header.Get("Content-Encoding") != "gzip" {
		t.Fatal("Setup failed: encoding header missing")
	}

	// Run decoder
	decoder.Response(f)

	// Verify after
	if f.Response.Header.Get("Content-Encoding") != "" {
		t.Error("Content-Encoding header not removed")
	}
	if string(f.Response.Body) != string(original) {
		t.Errorf("Body not decoded. Got: %s, Want: %s", f.Response.Body, original)
	}
    // Verify Content-Length updated (if logic does it, ReplaceToDecodedBody does)
    // Checking internal proxy method behavior via addon test
}
