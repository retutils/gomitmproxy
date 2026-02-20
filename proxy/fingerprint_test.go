package proxy

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIntegration_TlsFingerprint(t *testing.T) {
	// 1. Upstream HTTPS Server
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello fingerprint"))
	}))
	defer upstream.Close()

	// 2. Setup Proxy with Fingerprint
	opts := &Options{
		Addr:           ":9086",
		SslInsecure:    true,
		TlsFingerprint: "chrome", // Enable Chrome fingerprint
	}

	p, err := NewProxy(opts)
	if err != nil {
		t.Fatalf("NewProxy error: %v", err)
	}

	go p.Start()
	defer p.Close()
	time.Sleep(500 * time.Millisecond) // Wait for start

	// 3. Client
	proxyUrl, _ := url.Parse("http://127.0.0.1:9086")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Trust the proxy's MITM cert
			},
		},
	}

	// 4. Request
	resp, err := client.Get(upstream.URL)
	if err != nil {
		t.Fatalf("Client get with fingerprint error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello fingerprint" {
		t.Errorf("Want 'hello fingerprint', got '%s'", string(body))
	}
}

func TestIntegration_ClientFingerprint(t *testing.T) {
	// 1. Upstream HTTPS Server
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello client fingerprint"))
	}))
	defer upstream.Close()

	// 2. Setup Proxy with 'client' Fingerprint
	opts := &Options{
		Addr:           ":9087",
		SslInsecure:    true,
		TlsFingerprint: "client", // Copy client fingerprint
	}

	p, err := NewProxy(opts)
	if err != nil {
		t.Fatalf("NewProxy error: %v", err)
	}

	go p.Start()
	defer p.Close()
	time.Sleep(500 * time.Millisecond) // Wait for start

	// 3. Client
	proxyUrl, _ := url.Parse("http://127.0.0.1:9087")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// 4. Request
	resp, err := client.Get(upstream.URL)
	if err != nil {
		t.Fatalf("Client get with client fingerprint error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello client fingerprint" {
		t.Errorf("Want 'hello client fingerprint', got '%s'", string(body))
	}
}

func TestIntegration_FingerprintSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	fpPath := filepath.Join(tmpDir, "saved_fp.json")

	// Phase 1: Save Fingerprint
	{
		upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer upstream.Close()

		opts := &Options{
			Addr:            ":9088",
			SslInsecure:     true,
			FingerprintSave: fpPath,
		}
		p, err := NewProxy(opts)
		if err != nil {
			t.Fatalf("NewProxy error: %v", err)
		}
		go p.Start()
		defer p.Close()
		time.Sleep(500 * time.Millisecond)

		proxyUrl, _ := url.Parse("http://127.0.0.1:9088")
		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		client.Get(upstream.URL)

		// Check file exists
		if _, err := os.Stat(fpPath); os.IsNotExist(err) {
			t.Fatalf("Fingerprint file not saved at %s", fpPath)
		}
	}

	// Phase 2: Load Fingerprint
	{
		upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("hello saved fingerprint"))
		}))
		defer upstream.Close()

		opts := &Options{
			Addr:           ":9089",
			SslInsecure:    true,
			TlsFingerprint: fpPath, // Load from saved file
		}
		p, err := NewProxy(opts)
		if err != nil {
			t.Fatalf("NewProxy error: %v", err)
		}
		go p.Start()
		defer p.Close()
		time.Sleep(500 * time.Millisecond)

		proxyUrl, _ := url.Parse("http://127.0.0.1:9089")
		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		resp, err := client.Get(upstream.URL)
		if err != nil {
			t.Fatalf("Client get with saved fingerprint error: %v", err)
		}
		defer resp.Body.Close()
		
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "hello saved fingerprint" {
			t.Errorf("Want 'hello saved fingerprint', got '%s'", string(body))
		}
	}
}

func TestListFingerprints(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := FingerprintDir
	FingerprintDir = tmpDir
	defer func() { FingerprintDir = origDir }()

	SaveFingerprint("fp1", &Fingerprint{Name: "fp1"})
	SaveFingerprint("fp2", &Fingerprint{Name: "fp2"})
	os.WriteFile(filepath.Join(tmpDir, "not_json.txt"), []byte("..."), 0644)

	names, err := ListFingerprints()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Errorf("Expected 2 fingerprints, got %d", len(names))
	}
}

func TestSaveFingerprint_Error(t *testing.T) {
	// 1. Permission denied
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "readonly")
	os.Mkdir(path, 0444)
	defer os.Chmod(path, 0755)
	
	err := SaveFingerprint(filepath.Join(path, "fp.json"), &Fingerprint{})
	if err == nil {
		t.Error("Expected error for non-writable path")
	}
}

func TestLoadFingerprint_Error(t *testing.T) {
	// 1. Invalid JSON
	tmpFile, _ := os.CreateTemp("", "invalid.json")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("{invalid}")
	tmpFile.Close()
	
	_, err := LoadFingerprint(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// 2. Non-existent in default dir
	_, err = LoadFingerprint("completely-missing-fp")
	if err == nil {
		t.Error("Expected error for missing fingerprint")
	}
}
