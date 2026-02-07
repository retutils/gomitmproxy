package addon

import (
	"net/url"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestMapFrom_Match(t *testing.T) {
	tests := []struct {
		name     string
		mf       *MapFrom
		req      *proxy.Request
		expected bool
	}{
		{
			name: "Match logic all fields",
			mf: &MapFrom{
				Protocol: "https",
				Host:     "example.com",
				Method:   []string{"GET", "POST"},
				Path:     "/foo",
			},
			req: &proxy.Request{
				Method: "GET",
				URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/foo"},
			},
			expected: true,
		},
		{
			name: "Mismatch logic protocol",
			mf: &MapFrom{Protocol: "https"},
			req: &proxy.Request{
				Method: "GET",
				URL:    &url.URL{Scheme: "http", Host: "example.com", Path: "/foo"},
			},
			expected: false,
		},
		{
			name: "Mismatch logic host",
			mf: &MapFrom{Host: "example.com"},
			req: &proxy.Request{
				Method: "GET",
				URL:    &url.URL{Scheme: "https", Host: "other.com", Path: "/foo"},
			},
			expected: false,
		},
		{
			name: "Mismatch logic method",
			mf: &MapFrom{Method: []string{"POST"}},
			req: &proxy.Request{
				Method: "GET",
				URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/foo"},
			},
			expected: false,
		},
		{
			name: "Mismatch logic path",
			mf: &MapFrom{Path: "/bar"},
			req: &proxy.Request{
				Method: "GET",
				URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/foo"},
			},
			expected: false,
		},
		{
			name: "Match logic wildcard path",
			mf: &MapFrom{Path: "/foo/*"},
			req: &proxy.Request{
				Method: "GET",
				URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/foo/bar"},
			},
			expected: true,
		},
        {
			name: "Empty criteria matches all",
			mf: &MapFrom{},
			req: &proxy.Request{
				Method: "GET",
				URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/foo"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mf.Match(tt.req); got != tt.expected {
				t.Errorf("MapFrom.Match() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMapFrom_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mf      *MapFrom
		wantErr bool
	}{
		{
			name:    "Valid protocol http",
			mf:      &MapFrom{Protocol: "http"},
			wantErr: false,
		},
		{
			name:    "Valid protocol https",
			mf:      &MapFrom{Protocol: "https"},
			wantErr: false,
		},
		{
			name:    "Valid empty protocol",
			mf:      &MapFrom{Protocol: ""},
			wantErr: false,
		},
		{
			name:    "Invalid protocol",
			mf:      &MapFrom{Protocol: "ftp"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.mf.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("MapFrom.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
