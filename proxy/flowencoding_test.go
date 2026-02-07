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

func TestIsTextContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"text/plain", true},
		{"application/json", true},
		{"application/javascript", true},
		{"image/png", false},
		{"", false},
	}

	for _, tt := range tests {
		resp := &Response{Header: make(http.Header)}
		if tt.contentType != "" {
			resp.Header.Set("Content-Type", tt.contentType)
		}
		if got := resp.IsTextContentType(); got != tt.want {
			t.Errorf("IsTextContentType(%q) = %v; want %v", tt.contentType, got, tt.want)
		}
	}
}

func TestDecodedBody(t *testing.T) {
    original := []byte("hello world")
    
    // Gzip
    var gzipBuf bytes.Buffer
    gw := gzip.NewWriter(&gzipBuf)
    gw.Write(original)
    gw.Close()
    
    // Deflate
    var deflateBuf bytes.Buffer
    dw, _ := flate.NewWriter(&deflateBuf, flate.DefaultCompression)
    dw.Write(original)
    dw.Close()
    
    // Brotli
    var brBuf bytes.Buffer
    bw := brotli.NewWriter(&brBuf)
    bw.Write(original)
    bw.Close()
    
    // Zstd
    var zstdBuf bytes.Buffer
    zw, _ := zstd.NewWriter(&zstdBuf)
    zw.Write(original)
    zw.Close()
    
    tests := []struct {
        encoding string
        body []byte
        wantErr bool
    }{
        {"gzip", gzipBuf.Bytes(), false},
        {"deflate", deflateBuf.Bytes(), false},
        {"br", brBuf.Bytes(), false},
        {"zstd", zstdBuf.Bytes(), false},
        {"identity", original, false},
        {"", original, false},
        {"unknown", original, true},
    }
    
    for _, tt := range tests {
        // Test Request
        req := &Request{Header: make(http.Header), Body: tt.body}
        req.Header.Set("Content-Encoding", tt.encoding)
        gotReq, err := req.DecodedBody()
        if (err != nil) != tt.wantErr {
            t.Errorf("Request.DecodedBody(%s) error = %v, wantErr %v", tt.encoding, err, tt.wantErr)
        }
        if !tt.wantErr && string(gotReq) != string(original) {
            t.Errorf("Request.DecodedBody(%s) = %s, want %s", tt.encoding, gotReq, original)
        }
        
        // Test Response
        resp := &Response{Header: make(http.Header), Body: tt.body}
        resp.Header.Set("Content-Encoding", tt.encoding)
        gotResp, err := resp.DecodedBody()
        if (err != nil) != tt.wantErr {
            t.Errorf("Response.DecodedBody(%s) error = %v, wantErr %v", tt.encoding, err, tt.wantErr)
        }
        if !tt.wantErr && string(gotResp) != string(original) {
             t.Errorf("Response.DecodedBody(%s) = %s, want %s", tt.encoding, gotResp, original)
        }
    }
    
    // Test Response.ReplaceToDecodedBody
    resp := &Response{Header: make(http.Header), Body: gzipBuf.Bytes()}
    resp.Header.Set("Content-Encoding", "gzip")
    resp.ReplaceToDecodedBody()
    
    if string(resp.Body) != string(original) {
        t.Errorf("ReplaceToDecodedBody failed, got %s", resp.Body)
    }
    if resp.Header.Get("Content-Encoding") != "" {
        t.Error("Content-Encoding header should be removed")
    }
}

func TestDecodeError(t *testing.T) {
    // Test invalid gzip
    _, err := decode("gzip", []byte("invalid data"))
    if err == nil {
        t.Error("expected error for invalid gzip")
    }
    
    // Test invalid zstd
    _, err = decode("zstd", []byte("invalid data"))
    if err == nil {
        t.Error("expected error for invalid zstd")
    }
}
