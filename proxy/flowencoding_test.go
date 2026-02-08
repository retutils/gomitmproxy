package proxy

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"net/http"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

func TestFlowEncoding_DecodedBody_Request(t *testing.T) {
	// Gzip encoding
	var gzipBuf bytes.Buffer
	gw := gzip.NewWriter(&gzipBuf)
	gw.Write([]byte("gzip data"))
	gw.Close()

	req := &Request{
		Header: http.Header{"Content-Encoding": []string{"gzip"}},
		Body:   gzipBuf.Bytes(),
	}

	body, err := req.DecodedBody()
	if err != nil {
		t.Fatalf("DecodedBody error: %v", err)
	}
	if string(body) != "gzip data" {
		t.Errorf("Expected 'gzip data', got '%s'", body)
	}

	// No body
	req.Body = nil
	body, _ = req.DecodedBody()
	if body != nil {
		t.Error("Expected nil body")
	}

	// Identity encoding
	req.Body = []byte("raw")
	req.Header.Set("Content-Encoding", "identity")
	body, _ = req.DecodedBody()
	if string(body) != "raw" {
		t.Error("Expected raw body")
	}
    
    // Unsupported encoding
    req.Header.Set("Content-Encoding", "unknown")
    _, err = req.DecodedBody()
    if err == nil {
        t.Error("Expected error for unknown encoding")
    }
}

func TestFlowEncoding_DecodedBody_Response(t *testing.T) {
	// Brotli (br)
	var brBuf bytes.Buffer
	bw := brotli.NewWriter(&brBuf)
	bw.Write([]byte("br data"))
	bw.Close()

	resp := &Response{
		Header: http.Header{"Content-Encoding": []string{"br"}},
		Body:   brBuf.Bytes(),
	}

	body, err := resp.DecodedBody()
	if err != nil {
		t.Fatalf("DecodedBody br error: %v", err)
	}
	if string(body) != "br data" {
		t.Errorf("Expected 'br data', got '%s'", body)
	}
    
    // Deflate
    var flateBuf bytes.Buffer
    fw, _ := flate.NewWriter(&flateBuf, flate.DefaultCompression)
    fw.Write([]byte("deflate data"))
    fw.Close()
    
    resp.Header.Set("Content-Encoding", "deflate")
    resp.Body = flateBuf.Bytes()
    body, err = resp.DecodedBody()
    if err != nil {
        t.Fatalf("DecodedBody deflate error: %v", err)
    }
    if string(body) != "deflate data" {
        t.Errorf("Expected 'deflate data', got '%s'", body)
    }

	// Zstd
	var zstdBuf bytes.Buffer
	zw, _ := zstd.NewWriter(&zstdBuf)
	zw.Write([]byte("zstd data"))
	zw.Close()

	resp.Header.Set("Content-Encoding", "zstd")
	resp.Body = zstdBuf.Bytes()
	body, err = resp.DecodedBody()
	if err != nil {
		t.Fatalf("DecodedBody zstd error: %v", err)
	}
	if string(body) != "zstd data" {
		t.Errorf("Expected 'zstd data', got '%s'", body)
	}
	
	// Test Empty Body
	respEmpty := &Response{Body: nil}
	body, _ = respEmpty.DecodedBody()
	if body != nil {
		t.Error("Expected nil body")
	}
	
	// Test Default Encoding (empty string)
	respDef := &Response{
		Header: http.Header{},
		Body:   []byte("raw"),
	}
	body, _ = respDef.DecodedBody()
	if string(body) != "raw" {
		t.Error("Expected raw body (empty encoding)")
	}
	
	// Test Identity Encoding
	respId := &Response{
		Header: http.Header{"Content-Encoding": []string{"identity"}},
		Body:   []byte("raw"),
	}
	body, _ = respId.DecodedBody()
	if string(body) != "raw" {
		t.Error("Expected raw body (identity)")
	}
}

func TestFlowEncoding_ReplaceToDecodedBody(t *testing.T) {
	var gzipBuf bytes.Buffer
	gw := gzip.NewWriter(&gzipBuf)
	gw.Write([]byte("decodable"))
	gw.Close()

	resp := &Response{
		Header: http.Header{
			"Content-Encoding":  []string{"gzip"},
			"Transfer-Encoding": []string{"chunked"},
		},
		Body: gzipBuf.Bytes(),
	}

	resp.ReplaceToDecodedBody()

	if string(resp.Body) != "decodable" {
		t.Errorf("Body not decoded")
	}
	if resp.Header.Get("Content-Encoding") != "" {
		t.Errorf("Content-Encoding not removed")
	}
	if resp.Header.Get("Content-Length") != "9" { // len("decodable")
		t.Errorf("Content-Length not set correctly")
	}
	if resp.Header.Get("Transfer-Encoding") != "" {
		t.Errorf("Transfer-Encoding not removed")
	}
}

func TestFlowEncoding_IsTextContentType(t *testing.T) {
	resp := &Response{Header: http.Header{}}
	if resp.IsTextContentType() {
		t.Error("Empty header should not be text")
	}
	
	resp.Header.Set("Content-Type", "application/json")
	if !resp.IsTextContentType() {
		t.Error("application/json should be text")
	}
	
	resp.Header.Set("Content-Type", "image/png")
	if resp.IsTextContentType() {
		t.Error("image/png should not be text")
	}
}

func TestFlowEncoding_DecodeErrors(t *testing.T) {
	// Gzip error
	_, err := decode("gzip", []byte("invalid"))
	if err == nil {
		t.Error("Expected gzip error")
	}
	
	// Br error (brotli reader usually doesn't error on creation, but on read)
	// mock read error?
	
	// Deflate error
	_, err = decode("deflate", []byte("invalid"))
	if err == nil {
		t.Error("Expected deflate error")
	}
	
	// Zstd error
	_, err = decode("zstd", []byte("invalid"))
	if err == nil {
		t.Error("Expected zstd error")
	}
}
