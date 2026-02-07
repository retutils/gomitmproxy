package proxy

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
    
    uuid "github.com/satori/go.uuid"
)

func TestFlow_MarshalJSON(t *testing.T) {
    f := NewFlow()
    f.Id = uuid.NewV4()
    f.Request = NewRequest(&http.Request{
        Method: "GET",
        URL: &url.URL{
            Scheme: "http",
            Host:   "example.com",
            Path:   "/foo",
        },
        Header: make(http.Header),
    })
    
    // Marshal
    data, err := json.Marshal(f)
    if err != nil {
        t.Fatal(err)
    }
    
    // Unmarshal?
    // func (f *Flow) UnmarshalJSON(b []byte) error is defined?
    // flow.go:51: UnmarshalJSON 0.0%
    
    var f2 Flow
    err = json.Unmarshal(data, &f2)
    if err != nil {
        t.Fatal(err)
    }
    
    if f2.Id != f.Id {
        t.Errorf("ID mismatch: %v != %v", f2.Id, f.Id)
    }
    if f2.Request.Method != "GET" {
        t.Errorf("Method mismatch: %v", f2.Request.Method)
    }
}
