package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeConfigs(t *testing.T) {
    fileConfig := &Config{
        Addr: ":8080",
        SslInsecure: false,
    }
    cliConfig := &Config{
        Addr: ":9090",
        SslInsecure: true,
        Debug: 1,
    }
    
    merged := mergeConfigs(fileConfig, cliConfig)
    if merged.Addr != ":9090" {
        t.Errorf("Addr not overridden")
    }
    if !merged.SslInsecure {
        t.Errorf("SslInsecure not overridden")
    }
    if merged.Debug != 1 {
        t.Errorf("Debug not merged")
    }
}

func TestLoadConfigFromFile(t *testing.T) {
    tmpDir := t.TempDir()
    path := filepath.Join(tmpDir, "config.json")
    
    cfg := &Config{Addr: ":1234"}
    data, _ := json.Marshal(cfg)
    os.WriteFile(path, data, 0644)
    
    loaded, err := loadConfigFromFile(path)
    if err != nil {
        t.Fatal(err)
    }
    if loaded.Addr != ":1234" {
        t.Errorf("expected :1234, got %s", loaded.Addr)
    }
}

func TestArrayValue(t *testing.T) {
    var av arrayValue
    av.Set("foo")
    av.Set("bar")
    if len(av) != 2 {
        t.Errorf("expected len 2")
    }
    if av.String() != "[foo bar]" {
        t.Errorf("unexpected string: %s", av.String())
    }
}
