package helper

import (
	"bytes"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestCopy(t *testing.T) {
	src := bytes.NewBufferString("test data")
	dst := &bytes.Buffer{}
	n, err := Copy(dst, src)
	if err != nil {
		t.Errorf("Copy failed: %v", err)
	}
	if n != 9 {
		t.Errorf("Expected 9 bytes written, got %d", n)
	}
	if dst.String() != "test data" {
		t.Errorf("Expected 'test data', got %s", dst.String())
	}
}

func TestReaderToBuffer(t *testing.T) {
	t.Run("BelowLimit", func(t *testing.T) {
		src := bytes.NewBufferString("test")
		buf, r, err := ReaderToBuffer(src, 10)
		if err != nil {
			t.Errorf("ReaderToBuffer failed: %v", err)
		}
		if string(buf) != "test" {
			t.Errorf("Expected 'test', got %s", string(buf))
		}
		if r != nil {
			t.Errorf("Expected nil reader, got %v", r)
		}
	})

	t.Run("AtLimit", func(t *testing.T) {
		src := bytes.NewBufferString("testdata")
		buf, r, err := ReaderToBuffer(src, 8)
		if err != nil {
			t.Errorf("ReaderToBuffer failed: %v", err)
		}
		if buf != nil {
			t.Errorf("Expected nil buffer, got %s", string(buf))
		}
		if r == nil {
			t.Fatal("Expected new reader, got nil")
		}
		data, _ := io.ReadAll(r)
		if string(data) != "testdata" {
			t.Errorf("Expected 'testdata', got %s", string(data))
		}
	})
}

func TestNewStructFromFile(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}
	f, _ := os.CreateTemp("", "test.json")
	defer os.Remove(f.Name())
	f.WriteString(`{"name": "test"}`)
	f.Close()

	var v TestStruct
	err := NewStructFromFile(f.Name(), &v)
	if err != nil {
		t.Errorf("NewStructFromFile failed: %v", err)
	}
	if v.Name != "test" {
		t.Errorf("Expected 'test', got %s", v.Name)
	}

	// Test non-existent file
	err = NewStructFromFile("non_existent_file", &v)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test invalid JSON
	f2, _ := os.CreateTemp("", "invalid.json")
	defer os.Remove(f2.Name())
	f2.WriteString(`{invalid}`)
	f2.Close()
	err = NewStructFromFile(f2.Name(), &v)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestCanonicalAddr(t *testing.T) {
	tests := []struct {
		urlStr   string
		expected string
	}{
		{"http://example.com", "example.com:80"},
		{"https://example.com", "example.com:443"},
		{"http://example.com:8080", "example.com:8080"},
		{"socks5://example.com", "example.com:1080"},
	}
	for _, tt := range tests {
		u, _ := url.Parse(tt.urlStr)
		if got := CanonicalAddr(u); got != tt.expected {
			t.Errorf("CanonicalAddr(%s) = %s, want %s", tt.urlStr, got, tt.expected)
		}
	}
}

func TestIsTls(t *testing.T) {
	tests := []struct {
		buf      []byte
		expected bool
	}{
		{[]byte{0x16, 0x03, 0x01}, true},
		{[]byte{0x16, 0x03, 0x03}, true},
		{[]byte{0x16, 0x03, 0x04}, false},
		{[]byte{0x15, 0x03, 0x01}, false},
		{[]byte{0x16, 0x02, 0x01}, false},
	}
	for _, tt := range tests {
		if got := IsTls(tt.buf); got != tt.expected {
			t.Errorf("IsTls(%v) = %v, want %v", tt.buf, got, tt.expected)
		}
	}
}

func TestResponseCheck(t *testing.T) {
	recorder := httptest.NewRecorder()
	rc := NewResponseCheck(recorder)
	
	check, ok := rc.(*ResponseCheck)
	if !ok {
		t.Fatal("Expected *ResponseCheck type")
	}
	
	if check.Wrote {
		t.Error("Expected Wrote to be false initially")
	}
	
	rc.WriteHeader(200)
	if !check.Wrote {
		t.Error("Expected Wrote to be true after WriteHeader")
	}
	if recorder.Code != 200 {
		t.Errorf("Expected code 200, got %d", recorder.Code)
	}
	
	rc.Write([]byte("ok"))
	if recorder.Body.String() != "ok" {
		t.Errorf("Expected 'ok', got %s", recorder.Body.String())
	}
}
