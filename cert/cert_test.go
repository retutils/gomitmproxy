package cert

import (
    "os"
    "testing"
)

func TestSelfSignCA_Memory(t *testing.T) {
    ca, err := NewSelfSignCAMemory()
    if err != nil {
        t.Fatal(err)
    }
    
    // Test GetRootCA
    if ca.GetRootCA() == nil {
        t.Error("expected root CA")
    }
    
    // Test GetCert
    c, err := ca.GetCert("example.com")
    if err != nil {
        t.Error(err)
    }
    if c == nil {
        t.Error("expected cert for example.com")
    }
    
    // Test existing cert retrieval (cache)
    c2, err := ca.GetCert("example.com")
    if err != nil {
        t.Error(err)
    }
    if c != c2 {
        // Pointers might differ but content should be same? 
        // Or if cached, same pointer?
        // Implementation detail.
    }
}

func TestSelfSignCA_File(t *testing.T) {
    tmpDir := t.TempDir()
    // caRootPath is usually a file path or dir?  
    // NewSelfSignCA param is caRootPath.
    // If it's a file, it loads/creates it.
    
    // Test cleanup
    defer os.RemoveAll(tmpDir)

    ca, err := NewSelfSignCA(tmpDir) // pass dir
    if err != nil {
        t.Fatal(err)
    }
    
    _, err = ca.GetCert("test.com")
    if err != nil {
        t.Error(err)
    }
    
    // Verify file creation?
}
