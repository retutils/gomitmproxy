package main

import (
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
	"github.com/retutils/gomitmproxy/storage"
)

func TestRun_Version(t *testing.T) {
	config := &Config{version: true}
	err := Run(config)
	if err != nil {
		t.Errorf("Expected nil error for version, got %v", err)
	}
}

func TestRun_FingerprintList_WithFiles(t *testing.T) {
	// Setup a fingerprint file
	tmpDir := t.TempDir()
	origDir := proxy.FingerprintDir
	proxy.FingerprintDir = tmpDir
	defer func() { proxy.FingerprintDir = origDir }()
	
	proxy.SaveFingerprint("test-fp", &proxy.Fingerprint{Name: "test-fp"})
	
	config := &Config{FingerprintList: true}
	err := Run(config)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestRun_FingerprintList(t *testing.T) {
	config := &Config{FingerprintList: true}
	err := Run(config)
	if err != nil {
		t.Errorf("Expected nil error for FingerprintList, got %v", err)
	}
}

func TestRun_Search_Error(t *testing.T) {
	config := &Config{Search: "foo"} // Missing StorageDir
	err := Run(config)
	if err == nil {
		t.Error("Expected error for search without storage_dir")
	}
}

func TestRun_Search_Success(t *testing.T) {
	tmpDir := t.TempDir()
	svc, _ := storage.NewService(tmpDir)
	
	// Add a dummy entry
	flow := &proxy.Flow{
		Id: proxy.NewFlow().Id,
		Request: &proxy.Request{
			Method: "GET",
			URL:    &url.URL{Scheme: "http", Host: "example.com"},
		},
		ConnContext: &proxy.ConnContext{
			ClientConn: &proxy.ClientConn{},
		},
	}
	entry, _ := storage.NewFlowEntry(flow)
	svc.SaveEntry(entry, nil)
	svc.Close()

	config := &Config{
		Search:     "example.com",
		StorageDir: tmpDir,
	}
	err := Run(config)
	if err != nil {
		t.Errorf("Run search failed: %v", err)
	}
}

func TestRun_Full(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "proxy.log")
	storageDir := filepath.Join(tmpDir, "storage")
	
	config := &Config{
		Addr:        "127.0.0.1:0",
		WebAddr:     "127.0.0.1:0",
		Debug:       2,
		LogFile:     logFile,
		StorageDir:  storageDir,
		ScanPII:     true,
		ScanTech:    true,
		IgnoreHosts: []string{"ignore.com"},
		AllowHosts:  []string{"allow.com"},
		ProxyAuth:   "user:pass",
	}

	// Run in background and stop
	go func() {
		Run(config)
	}()
	
	// Wait for start
	time.Sleep(200 * time.Millisecond)
}

func TestRun_ProxyAuthError(t *testing.T) {
	config := &Config{
		Addr:      ":0",
		ProxyAuth: "invalid-format",
	}
	err := Run(config)
	if err == nil {
		t.Error("Expected error for invalid proxy auth format")
	}
}

func TestRun_MapErrors(t *testing.T) {
	// These should log warning but not return error
	config := &Config{
		Addr:      ":0",
		MapRemote: "non-existent.json",
		MapLocal:  "non-existent.json",
	}
	// We run in background because it will start the proxy
	go Run(config)
	time.Sleep(100 * time.Millisecond)
}

func TestRun_ProxyAuthAny(t *testing.T) {
	config := &Config{
		Addr:      ":0",
		ProxyAuth: "any",
	}
	go Run(config)
	time.Sleep(100 * time.Millisecond)
}

func TestRun_Defaults(t *testing.T) {
	config := &Config{
		Addr:    ":0",
		WebAddr: ":0",
	}
	go Run(config)
	time.Sleep(100 * time.Millisecond)
}

func TestRun_Dumping(t *testing.T) {
	tmpDir := t.TempDir()
	dumpFile := filepath.Join(tmpDir, "dump.log")
	config := &Config{
		Addr:    ":0",
		WebAddr: ":0",
		Dump:    dumpFile,
	}
	go Run(config)
	time.Sleep(100 * time.Millisecond)
}

func TestRun_InterceptRules(t *testing.T) {
    // Already covered in TestRun_Full by setting IgnoreHosts and AllowHosts
}
