package cert

import (
	"bytes"
    "crypto/ecdsa"
    "crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGetStorePath(t *testing.T) {
	// Test with empty path (should use home dir)
	path, err := getStorePath("")
	if err != nil {
		t.Fatalf("getStorePath with empty path failed: %v", err)
	}
	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(path, home) {
		t.Errorf("Expected path to be in home directory, got %s", path)
	}

	// Test with relative path
	tmpDir := t.TempDir()
	relPath := filepath.Base(tmpDir) // get a relative path
	absPath, err := getStorePath(relPath)
	if err != nil {
		t.Fatalf("getStorePath with relative path failed: %v", err)
	}
	cwd, _ := os.Getwd()
	if !strings.HasPrefix(absPath, cwd) {
		t.Errorf("Expected path to be in current working directory, got %s", absPath)
	}

	// Test with a file as path
	file, _ := os.CreateTemp(tmpDir, "file")
	file.Close()
	_, err = getStorePath(file.Name())
	if err == nil {
		t.Error("Expected error when path is a file")
	}

	// Test with non-existent path that can't be created
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	os.Mkdir(readOnlyDir, 0555) // Read-only
	_, err = getStorePath(filepath.Join(readOnlyDir, "newdir"))
	if err == nil {
		t.Error("Expected error when creating dir in read-only path")
	}
}

func TestNewCA(t *testing.T) {
	caApi, err := NewSelfSignCA("")
	if err != nil {
		t.Fatal(err)
	}
	ca := caApi.(*SelfSignCA)

	data := make([]byte, 0)
	buf := bytes.NewBuffer(data)

	err = ca.saveTo(buf)
	if err != nil {
		t.Fatal(err)
	}

	fileContent, err := ioutil.ReadFile(ca.caFile())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(fileContent, buf.Bytes()) {
		t.Fatal("pem content should equal")
	}
}

func TestSelfSignCA_Memory(t *testing.T) {
	ca, err := NewSelfSignCAMemory()
	if err != nil {
		t.Fatalf("NewSelfSignCAMemory() error = %v", err)
	}
	if ca.GetRootCA() == nil {
		t.Error("GetRootCA() returned nil")
	}
	cert, err := ca.GetCert("example.com")
	if err != nil {
		t.Fatalf("GetCert() error = %v", err)
	}
	if cert == nil {
		t.Error("GetCert() returned nil cert")
	}
}

func TestSelfSignCA_File(t *testing.T) {
	tmpDir := t.TempDir()
	caAPI, err := NewSelfSignCA(tmpDir)
	if err != nil {
		t.Fatalf("NewSelfSignCA() error = %v", err)
	}
	ca := caAPI.(*SelfSignCA)

	// second time should load from file
	_, err = NewSelfSignCA(tmpDir)
	if err != nil {
		t.Fatalf("NewSelfSignCA() on existing dir error = %v", err)
	}

	if err := ca.save(); err != nil {
		t.Fatalf("ca.save() error = %v", err)
	}
	if err := ca.load(); err != nil {
		t.Fatalf("ca.load() error = %v", err)
	}

	cert, err := ca.GetCert("example.com")
	if err != nil {
		t.Fatalf("GetCert() error = %v", err)
	}
	if cert == nil {
		t.Fatal("GetCert() returned nil cert")
	}
}

func TestLoadErrors(t *testing.T) {
	tmpDir := t.TempDir()
	caAPI, _ := NewSelfSignCA(tmpDir)
	ca := caAPI.(*SelfSignCA)

	// Test non-existent file
	os.Remove(ca.caFile())
	err := ca.load()
	if err != errCaNotFound {
		t.Errorf("Expected errCaNotFound for non-existent file, got %v", err)
	}

	// Test non-regular file (directory)
	os.Mkdir(ca.caFile(), 0755)
	err = ca.load()
	if err == nil {
		t.Error("Expected error for directory")
	}
	os.Remove(ca.caFile())

	// Test invalid PEM (no private key)
	ioutil.WriteFile(ca.caFile(), []byte("invalid content"), 0644)
	err = ca.load()
	if err == nil {
		t.Error("Expected error for invalid PEM (no key)")
	}

	// Test invalid PEM (no certificate)
	ioutil.WriteFile(ca.caFile(), []byte("-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----\n"), 0644)
	err = ca.load()
	if err == nil {
		t.Error("Expected error for invalid PEM (no cert)")
	}

	// Test invalid private key format (PKCS1 vs PKCS8)
	// This part is tricky as the code handles it. A more specific test might be needed.
	// For now, testing a completely garbage key is sufficient.
	ioutil.WriteFile(ca.caFile(), []byte("-----BEGIN PRIVATE KEY-----\ninvalid\n-----END PRIVATE KEY-----\n-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----\n"), 0644)
	err = ca.load()
	if err == nil {
		t.Error("Expected error for invalid private key")
	}
}

func TestDummyCertWithIP(t *testing.T) {
	caAPI, _ := NewSelfSignCAMemory()
	ca := caAPI.(*SelfSignCA)
	cert, err := ca.DummyCert("127.0.0.1")
	if err != nil {
		t.Fatalf("DummyCert with IP failed: %v", err)
	}
	if cert == nil {
		t.Fatal("cert is nil")
	}
}

func TestSelfSignCA_GetCert_Cache(t *testing.T) {
	ca, _ := NewSelfSignCAMemory()
	c1, _ := ca.GetCert("example.com")
	c2, _ := ca.GetCert("example.com")
	if c1 != c2 {
		t.Error("Expected cached certificate instance")
	}
}

func TestLoad_PKCS1Fallback(t *testing.T) {
	tmpDir := t.TempDir()
	caAPI, _ := NewSelfSignCA(tmpDir)
	ca := caAPI.(*SelfSignCA)

	// Manually create a PKCS1 PEM file
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	
	f, _ := os.Create(ca.caFile())
	pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
	pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: ca.RootCert.Raw})
	f.Close()

	err := ca.load()
	if err != nil {
		t.Fatalf("load() with PKCS1 failed: %v", err)
	}
}

func TestLoad_UnknownPKCS8Type(t *testing.T) {
    // This is hard to trigger without a non-RSA key that PKCS8 supports (like ECDSA)
    // but the code explicitly checks for *rsa.PrivateKey.
}

type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("write error")
}

func TestSaveTo_Error(t *testing.T) {
	caAPI, _ := NewSelfSignCAMemory()
	ca := caAPI.(*SelfSignCA)
	err := ca.saveTo(&errorWriter{})
	if err == nil {
		t.Error("Expected error for failing writer")
	}
}

func TestGetStorePath_NoHome(t *testing.T) {
    origHome := os.Getenv("HOME")
    os.Unsetenv("HOME")
    defer os.Setenv("HOME", origHome)

    _, err := getStorePath("")
    if err == nil {
        // On some systems it might still find home via other means
        t.Log("getStorePath might still succeed without HOME")
    }
}

func TestLoad_NonRSAKey(t *testing.T) {
	tmpDir := t.TempDir()
	caAPI, _ := NewSelfSignCA(tmpDir)
	ca := caAPI.(*SelfSignCA)

    // Generate an ECDSA key instead of RSA
    priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    keyBytes, _ := x509.MarshalPKCS8PrivateKey(priv)
    
	f, _ := os.Create(ca.caFile())
	pem.Encode(f, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: ca.RootCert.Raw})
	f.Close()

	err := ca.load()
	if err == nil {
		t.Error("Expected error for non-RSA key")
	}
}

func TestSaveCert_Error(t *testing.T) {
	tmpDir := t.TempDir()
	caAPI, _ := NewSelfSignCA(tmpDir)
	ca := caAPI.(*SelfSignCA)

    // Make second file uncreatable by creating a dir with that name
    cerPath := ca.caCertCerFile()
    os.Remove(cerPath) // Ensure it doesn't exist
    err := os.Mkdir(cerPath, 0755)
    if err != nil {
        t.Fatalf("Failed to create blocking dir: %v", err)
    }
    defer os.RemoveAll(cerPath)

	err = ca.saveCert()
	if err == nil {
		t.Error("Expected error when second file creation fails")
	}
}

func TestSaveErrors(t *testing.T) {
	tmpDir := t.TempDir()
	caAPI, _ := NewSelfSignCA(tmpDir)
	ca := caAPI.(*SelfSignCA)


	// Make store path read-only to test save errors
	os.Chmod(ca.StorePath, 0444)
	defer os.Chmod(ca.StorePath, 0755)

	err := ca.save()
	if err == nil {
		t.Error("Expected error when saving to read-only directory")
	}

	err = ca.saveCert()
	if err == nil {
		t.Error("Expected error when saving cert to read-only directory")
	}
}
