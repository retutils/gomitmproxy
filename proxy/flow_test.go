package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestFlow_Request_MarshalUnmarshal(t *testing.T) {
	u, _ := url.Parse("http://example.com/foo")
	req := &Request{
		Method: "GET",
		URL:    u,
		Proto:  "HTTP/1.1",
		Header: http.Header{"Content-Type": []string{"application/json"}},
	}
	
	bytes, err := req.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	
	var req2 Request
	err = json.Unmarshal(bytes, &req2)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	
	if req2.Method != req.Method {
		t.Errorf("Method mistmatch: %s != %s", req2.Method, req.Method)
	}
	if req2.URL.String() != req.URL.String() {
		t.Errorf("URL mistmatch: %s != %s", req2.URL.String(), req.URL.String())
	}
	if req2.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Header mismatch")
	}
}

func TestFlow_Request_UnmarshalErrors(t *testing.T) {
	var req Request
	
	// Invalid JSON
	err := json.Unmarshal([]byte(`{invalid`), &req)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	
	// Invalid URL type
	err = json.Unmarshal([]byte(`{"url": 123}`), &req)
	if err == nil || err.Error() != "url parse error" { // Check error string match or type
		t.Error("Expected url parse error")
	}

	// Invalid URL
	err = json.Unmarshal([]byte(`{"url": ":::"}`), &req) // Invalid URL char?
	// url.Parse might fail on this
	if err == nil {
		t.Error("Expected error on invalid URL string")
	}

	// Invalid Header type
	err = json.Unmarshal([]byte(`{"url": "http://a.com", "header": 123}`), &req)
	if err == nil {
		t.Error("Expected rawheader parse error")
	}
	
	// Invalid Header values type
	err = json.Unmarshal([]byte(`{"url": "http://a.com", "header": {"a": 1}}`), &req)
	if err == nil {
		t.Error("Expected header values parse error")
	}
	
	// Invalid Header value string type
	err = json.Unmarshal([]byte(`{"url": "http://a.com", "header": {"a": [1]}}`), &req)
	if err == nil {
		t.Error("Expected header value string parse error")
	}

    // Missing fields
    err = json.Unmarshal([]byte(`{}`), &req)
    if err == nil {
        t.Log("Unmarshal empty json returns nil err")
    }
}

func TestFlow_Request_Marshal_EmptyURL(t *testing.T) {
    req := &Request{}
    b, err := req.MarshalJSON()
    if err != nil { t.Fatal(err) }
    if !bytes.Contains(b, []byte(`"url":""`)) {
        t.Errorf("Expected empty url string, got %s", string(b))
    }
}

func TestFlow_Request_Raw(t *testing.T) {
	r := &Request{}
	req := &http.Request{}
	r.SetRaw(req)
	if r.Raw() != req {
		t.Error("SetRaw/Raw mismatch")
	}
}

func TestFlow_Done(t *testing.T) {
	f := NewFlow()
	select {
	case <-f.Done():
		t.Error("Done should ensure channel")
	default:
	}
	f.Finish()
	select {
	case <-f.Done():
	default:
		t.Error("Done should be closed")
	}
}

func TestFlow_MarshalJSON(t *testing.T) {
	f := NewFlow()
	f.Request = &Request{Method: "GET"}
	_, err := f.MarshalJSON() // Covers (*Flow).MarshalJSON
	if err != nil {
		t.Error("Flow MarshalJSON failed")
	}
}
