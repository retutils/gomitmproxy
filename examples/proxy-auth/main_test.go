package main

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestUserAuth_AuthEntrypAuth(t *testing.T) {
	auth := &UserAuth{Username: "user", Password: "pass"}
	
	tests := []struct {
		name    string
		header  string
		want    bool
		wantErr bool
	}{
		{"Empty", "", false, true},
		{"InvalidPrefix", "Bearer token", false, true},
		{"InvalidBase64", "Basic !!!", false, true},
		{"WrongFormat", "Basic dXNlcg==", false, true}, // user (no colon)
		{"WrongCreds", "Basic dXNlcjp3cm9uZw==", false, true}, // user:wrong
		{"Success", "Basic dXNlcjpwYXNz", true, false}, // user:pass
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			if tt.header != "" {
				req.Header.Set("Proxy-Authorization", tt.header)
			}
			got, err := auth.AuthEntrypAuth(nil, req)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRun(t *testing.T) {
	go Run()
	time.Sleep(100 * time.Millisecond)
}
