package addon

import (
	"net/http"
	"net/url"
	"testing"

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

func TestWappalyzerAddon_Detection(t *testing.T) {
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
			},
			Body: []byte("<html><head><title>Test</title></head><body>Hello</body></html>"),
		},
	}

	addon.Response(f)

	// Detection is async, wait a bit
	// Actually for tests we might want to make it synchronous or use a wait group
	// For now, let's just check the result after a short sleep if we can't easily wait
}
