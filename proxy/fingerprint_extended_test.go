package proxy

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprint_FileOperations(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "mitmproxy_fingerprint_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Override FingerprintDir
	origDir := FingerprintDir
	FingerprintDir = tmpDir
	defer func() { FingerprintDir = origDir }()
	
	// Prepare dummy fingerprint
	fp := &Fingerprint{
		Name:          "test_browser",
		CipherSuites:  []uint16{tls.TLS_AES_128_GCM_SHA256},
		ALPNProtocols: []string{"h2", "http/1.1"},
	}
	
	// Test SaveFingerprint (default dir)
	err = SaveFingerprint("test_browser", fp)
	if err != nil {
		t.Errorf("SaveFingerprint failed: %v", err)
	}
	
	// Check file exists
	expectedPath := filepath.Join(tmpDir, "test_browser.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Fingerprint file not created")
	}
	
	// Test ListFingerprints
	names, err := ListFingerprints()
	if err != nil {
		t.Errorf("ListFingerprints failed: %v", err)
	}
	if len(names) != 1 || names[0] != "test_browser" {
		t.Errorf("ListFingerprints expected [test_browser], got %v", names)
	}
	
	// Test LoadFingerprint (default dir)
	loadedFp, err := LoadFingerprint("test_browser")
	if err != nil {
		t.Errorf("LoadFingerprint failed: %v", err)
	}
	if loadedFp.Name != fp.Name {
		t.Errorf("Loaded fingerprint name mismatch")
	}
	
	// Test SaveFingerprint (absolute path)
	absPath := filepath.Join(tmpDir, "custom.json")
	err = SaveFingerprint(absPath, fp)
	if err != nil {
		t.Errorf("SaveFingerprint absolute failed: %v", err)
	}
	
	// Test LoadFingerprint (filename w/o extension in non-default dir? No, ReadFile needs full path usually or logic handles it)
	// LoadFingerprint logic:
	// 1. Try name as path
	// 2. Try default dir + name + .json
	// 3. Try default dir + name
	
	loadedFp2, err := LoadFingerprint(absPath)
	if err != nil {
		t.Errorf("LoadFingerprint absolute failed: %v", err)
	}
	if loadedFp2.Name != fp.Name {
		t.Errorf("Loaded absolute fp name mismatch")
	}
	
	// Test ensureDir with non-existent directory (should create it)
	nonExistentDir := filepath.Join(tmpDir, "brand_new_dir")
	FingerprintDir = nonExistentDir
	err = SaveFingerprint("new", fp)
	if err != nil {
		t.Errorf("SaveFingerprint failed creating new dir: %v", err)
	}
	if _, err := os.Stat(nonExistentDir); os.IsNotExist(err) {
		t.Error("Did not create directory")
	}

	// Test ensureDir failure (file as parent)
	// We run this last as it messes up things possibly
	fileAsParent := filepath.Join(tmpDir, "file_parent")
	os.WriteFile(fileAsParent, []byte("data"), 0644)
	FingerprintDir = filepath.Join(fileAsParent, "subdir")
	
	err = SaveFingerprint("fail_mkdir", fp)
	if err == nil {
		t.Error("Expected error from ensureDir (MkdirAll)")
	}
	
	// Restore valid dir for cleanup
	FingerprintDir = tmpDir
}

func TestFingerprint_ListFingerprints_EdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mitm_fp_list_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	origDir := FingerprintDir
	FingerprintDir = tmpDir
	defer func() { FingerprintDir = origDir }()
	
	// Case 1: Non-existent dir
	FingerprintDir = filepath.Join(tmpDir, "doesval_not_exist")
	names, err := ListFingerprints()
	if err != nil {
		t.Errorf("ListFingerprints on non-existent dir failed: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("Expected empty list, got %v", names)
	}
	
	// Create dir
	FingerprintDir = tmpDir
	
	// Case 2: Ignore directories
	os.Mkdir(filepath.Join(tmpDir, "ignored_subdir"), 0755)
	
	// Case 3: Ignore non-json files
	os.WriteFile(filepath.Join(tmpDir, "ignored.txt"), []byte("data"), 0644)
	
	// Case 4: Valid file
	os.WriteFile(filepath.Join(tmpDir, "valid.json"), []byte("{}"), 0644)
	
	names, err = ListFingerprints()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "valid" {
		t.Errorf("Expected [valid], got %v", names)
	}
	
	// Case 5: ReadDir error (permission denied)
	// Skip on windows?
	if os.Getuid() != 0 { // Skip if root
		os.Chmod(tmpDir, 0000)
		_, err = ListFingerprints()
		if err == nil {
			t.Error("Expected error on unreadable dir")
		}
		os.Chmod(tmpDir, 0755) // restore
	}
}

func TestFingerprint_FromClientHello(t *testing.T) {
	info := &tls.ClientHelloInfo{
		CipherSuites:     []uint16{0x1301},
		SupportedVersions: []uint16{0x0304},
		SupportedCurves:  []tls.CurveID{tls.X25519},
		SupportedPoints:  []uint8{0},
		SignatureSchemes: []tls.SignatureScheme{tls.ECDSAWithP256AndSHA256},
		SupportedProtos:  []string{"h2"},
	}
	fp := NewFingerprintFromClientHello("test", info)
	if fp.Name != "test" {
		t.Error("Name mismatch")
	}
	if len(fp.CipherSuites) != 1 || fp.CipherSuites[0] != 0x1301 {
		t.Error("CipherSuites copy fail")
	}
	
	spec := fp.ToSpec()
	if len(spec.CipherSuites) != 1 {
		t.Error("ToSpec cipher suites len mismatch")
	}
	// Check extensions
	if len(spec.Extensions) == 0 {
		t.Error("ToSpec missing extensions")
	}
}
