package addon

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
)

func TestPIIAddon(t *testing.T) {
	pii := NewPIIAddonStub()

	tests := []struct {
		name         string
		body         string
		contentType  string
		wantDetected bool
		wantTypes    []string
	}{
		{
			name:         "No PII",
			body:         "Hello world, this is a normal response.",
			contentType:  "text/plain",
			wantDetected: false,
		},
		{
			name:         "Email Detected",
			body:         "Contact us at support@example.com for help.",
			contentType:  "text/html",
			wantDetected: true,
			wantTypes:    []string{"Email"},
		},
		{
			name:         "IPv4 Detected",
			body:         "Server IP is 192.168.1.1 configured via DHCP.",
			contentType:  "application/json",
			wantDetected: true,
			wantTypes:    []string{"IPv4"},
		},
		{
			name:         "Keyword Detected",
			body:         "{\"api_key\": \"12345secret\"}",
			contentType:  "application/json",
			wantDetected: true,
			wantTypes:    []string{"SensitiveKeywords"},
		},
		{
			name:         "Multiple PII",
			body:         "User email: user@test.com, IP: 10.0.0.1, password: secret",
			contentType:  "text/plain",
			wantDetected: true,
			wantTypes:    []string{"Email", "IPv4", "SensitiveKeywords"},
		},
		{
			name:         "Ignore Binary",
			body:         "support@example.com",
			contentType:  "image/png",
			wantDetected: false,
		},
		{
			name:        "Decoding Failed",
			body:        "some encoded content", // Not valid GZIP
			contentType: "text/plain",           // Valid type so it enters decode check
			// We need to set Content-Encoding in test setup below, so we add a field or logic
			wantDetected: false,
		},
		{ // We need to handle Content-Encoding test case separately or extend struct
			name:         "Unknown Content Type",
			body:         "support@example.com",
			contentType:  "",
			wantDetected: false,
		},
		{
			name:         "Empty Body",
			body:         "",
			contentType:  "text/plain",
			wantDetected: false,
		},
		{
			name:         "XML Content Type",
			body:         "<email>test@example.com</email>",
			contentType:  "application/xml",
			wantDetected: true,
			wantTypes:    []string{"Email"},
		},
		{
			name:         "Javascript Content Type",
			body:         "var email = 'test@example.com';",
			contentType:  "application/javascript",
			wantDetected: true,
			wantTypes:    []string{"Email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse("http://example.com")
			f := &proxy.Flow{
				Request: &proxy.Request{
					Method: "GET",
					URL:    u,
				},
				Response: &proxy.Response{
					Header:     http.Header{},
					Body:       []byte(tt.body),
					StatusCode: 200,
				},
				Metadata: make(map[string]interface{}),
			}
			f.Response.Header.Set("Content-Type", tt.contentType)
			if tt.name == "Decoding Failed" {
				f.Response.Header.Set("Content-Encoding", "gzip")
			}

			pii.Response(f)

			if tt.wantDetected {
				meta, ok := f.Metadata["pii"]
				if !ok {
					t.Errorf("Expected PII metadata to be present, but got none")
					return
				}

				_, ok = meta.(bool)
				if !ok {
					t.Errorf("Expected metadata type bool, got %T", meta)
					return
				}
				// Since we only get true, we don't verify specific types anymore in Metadata,
				// but we could verify the log output if we hooked logger.
				// For now, just verifying it is flagged is enough per new requirement.
			} else {
				if _, ok := f.Metadata["pii"]; ok {
					t.Errorf("Expected no PII metadata, but got some")
				}
			}
		})
	}

	// Edge Case: Nil Response
	t.Run("Nil Response", func(t *testing.T) {
		f := &proxy.Flow{
			Request: &proxy.Request{},
			// Response is nil
		}
		pii.Response(f)
		// Should not panic or set metadata
		if f.Metadata != nil {
			if _, ok := f.Metadata["pii"]; ok {
				t.Errorf("Should not have PII metadata on nil response")
			}
		}
	})

	// Edge Case: Nil Body
	t.Run("Nil Body", func(t *testing.T) {
		f := &proxy.Flow{
			Request: &proxy.Request{},
			Response: &proxy.Response{
				Header: http.Header{},
				// Body is nil
			},
			Metadata: make(map[string]interface{}),
		}
		pii.Response(f)
		if _, ok := f.Metadata["pii"]; ok {
			t.Errorf("Should not have PII metadata on nil body")
		}
	})
}

// Helper to create addon for tests
func NewPIIAddonStub() *PIIAddon {
	return NewPIIAddon()
}
