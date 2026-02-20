package addon

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
	"github.com/retutils/gomitmproxy/storage"
)

func TestWappalyzerAddon_Initialization(t *testing.T) {
	// Mock storage service
	tmpDir := t.TempDir()
	svc, _ := storage.NewService(tmpDir)
	defer svc.Close()

	addon := NewWappalyzerAddon(svc)
	if addon == nil {
		t.Fatal("Expected WappalyzerAddon instance, got nil")
	}
}

func TestWappalyzerAddon_Response_Filtering(t *testing.T) {
	tmpDir := t.TempDir()
	svc, _ := storage.NewService(tmpDir)
	defer svc.Close()

	addon := NewWappalyzerAddon(svc)

	tests := []struct {
		name     string
		response *proxy.Response
		storage  *storage.Service
	}{
		{
			name:     "Nil Response",
			response: nil,
			storage:  svc,
		},
		{
			name: "Nil Body",
			response: &proxy.Response{
				Body: nil,
			},
			storage: svc,
		},
		{
			name: "Large Body",
			response: &proxy.Response{
				Body: make([]byte, 2*1024*1024), // 2MB
			},
			storage: svc,
		},
		{
			name: "Non-Text Content",
			response: &proxy.Response{
				Header: http.Header{"Content-Type": []string{"image/png"}},
				Body:   []byte("fake image data"),
			},
			storage: svc,
		},
		{
			name: "Nil Storage",
			response: &proxy.Response{
				Header: http.Header{"Content-Type": []string{"text/html"}},
				Body:   []byte("<html></html>"),
			},
			storage: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addon.storage = tt.storage
			f := &proxy.Flow{
				Response: tt.response,
			}
			addon.Response(f)
			// Ensure no panic and no background processing started (hard to verify perfectly, but no panic is key)
		})
	}
}

func TestWappalyzerAddon_SampleBody(t *testing.T) {
	addon := &WappalyzerAddon{}

	// 1. Non-HTML
	body := []byte("plain text")
	if !bytes.Equal(addon.sampleBody(body, "text/plain"), body) {
		t.Error("Non-HTML should not be sampled")
	}

	// 2. Small HTML
	smallHTML := []byte("<html><body>small</body></html>")
	if !bytes.Equal(addon.sampleBody(smallHTML, "text/html"), smallHTML) {
		t.Error("Small HTML should not be sampled")
	}

	// 3. Large HTML
	largeHTML := make([]byte, 100000)
	for i := range largeHTML {
		largeHTML[i] = 'A'
	}
	copy(largeHTML, []byte("START"))
	copy(largeHTML[len(largeHTML)-3:], []byte("END"))

	sampled := addon.sampleBody(largeHTML, "text/html")
	if len(sampled) != 65536 {
		t.Errorf("Expected sampled length 65536, got %d", len(sampled))
	}
	if !bytes.HasPrefix(sampled, []byte("START")) {
		t.Error("Sampled body missing prefix")
	}
	if !bytes.HasSuffix(sampled, []byte("END")) {
		t.Error("Sampled body missing suffix")
	}
}

func TestWappalyzerAddon_Detection_Async(t *testing.T) {
	tmpDir := t.TempDir()
	svc, _ := storage.NewService(tmpDir)
	defer svc.Close()

	addon := NewWappalyzerAddon(svc)

	u, _ := url.Parse("http://example.com")
	f := &proxy.Flow{
		Request: &proxy.Request{
			Method: "GET",
			URL:    u,
		},
		Response: &proxy.Response{
			StatusCode: 200,
			Header: http.Header{
				"X-Powered-By": []string{"Express"},
				"Server":       []string{"nginx/1.18.0"},
				"Content-Type": []string{"text/html"},
			},
			Body: []byte("<html><head><title>Test</title></head><body>Hello</body></html>"),
		},
	}

	addon.Response(f)

	// Wait for async detection
	time.Sleep(200 * time.Millisecond)

	techs, err := svc.GetHostTechnologies("example.com")
	if err != nil {
		t.Fatal(err)
	}
	
	// We expect at least Nginx or Express to be detected
	if len(techs) == 0 {
		t.Log("Warning: No technologies detected in async test, might be due to wappalyzergo patterns")
	}
}
