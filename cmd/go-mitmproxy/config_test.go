package main

import (
	"encoding/json"
	"flag"
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

func TestMergeConfigs_Full(t *testing.T) {
    fileConfig := &Config{
        Addr: ":1", WebAddr: ":2", SslInsecure: true, IgnoreHosts: []string{"i1"}, AllowHosts: []string{"a1"},
        CertPath: "c1", Debug: 1, Dump: "d1", DumpLevel: 1, Upstream: "u1", UpstreamCert: true,
        MapRemote: "mr1", MapLocal: "ml1", LogFile: "l1", TlsFingerprint: "tf1", FingerprintSave: "fs1",
        FingerprintList: true, StorageDir: "sd1", ScanPII: true, ScanTech: true,
    }
    cliConfig := &Config{
        Addr: ":11", WebAddr: ":22", SslInsecure: false, IgnoreHosts: []string{"i2"}, AllowHosts: []string{"a2"},
        CertPath: "c2", Debug: 2, Dump: "d2", DumpLevel: 2, Upstream: "u2", UpstreamCert: false,
        MapRemote: "mr2", MapLocal: "ml2", LogFile: "l2", TlsFingerprint: "tf2", FingerprintSave: "fs2",
        FingerprintList: false, StorageDir: "sd2", ScanPII: false,
    }
    
    merged := mergeConfigs(fileConfig, cliConfig)
    
    if merged.Addr != ":11" { t.Error("Addr") }
    if merged.WebAddr != ":22" { t.Error("WebAddr") }
    if !merged.SslInsecure { t.Error("SslInsecure should be true if file is true and cli is false (wait, logic check)") }
    
    if len(merged.IgnoreHosts) != 1 || merged.IgnoreHosts[0] != "i2" { t.Error("IgnoreHosts") }
    if merged.CertPath != "c2" { t.Error("CertPath") }
    if merged.Debug != 2 { t.Error("Debug") }
    if merged.Dump != "d2" { t.Error("Dump") }
    if merged.DumpLevel != 2 { t.Error("DumpLevel") }
    if merged.Upstream != "u2" { t.Error("Upstream") }
    if merged.UpstreamCert { t.Error("UpstreamCert should be false") }
    if merged.MapRemote != "mr2" { t.Error("MapRemote") }
    if merged.MapLocal != "ml2" { t.Error("MapLocal") }
    if merged.LogFile != "l2" { t.Error("LogFile") }
    if merged.TlsFingerprint != "tf2" { t.Error("TlsFingerprint") }
    if merged.FingerprintSave != "fs2" { t.Error("FingerprintSave") }
    if merged.StorageDir != "sd2" { t.Error("StorageDir") }
    if !merged.ScanTech { t.Error("ScanTech should be true from file") }
}

func TestMergeConfigs_Dns(t *testing.T) {
    fileConfig := &Config{
        DnsResolvers: []string{"1.1.1.1"},
        DnsRetries: 3,
    }
    cliConfig := &Config{
        DnsResolvers: []string{"8.8.8.8"},
        DnsRetries: 5,
    }
    merged := mergeConfigs(fileConfig, cliConfig)
    if merged.DnsResolvers[0] != "8.8.8.8" { t.Error("DnsResolvers") }
    if merged.DnsRetries != 5 { t.Error("DnsRetries") }
}

func TestLoadConfig_Error(t *testing.T) {
    _, err := loadConfigFromFile("non_existent.json")
    if err == nil {
        t.Error("Expected error for missing file")
    }
}

func TestDefineFlags(t *testing.T) {
	config := new(Config)
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	defineFlags(fs, config)

	args := []string{
		"-addr", ":1234",
		"-web_addr", ":5678",
		"-ssl_insecure",
		"-ignore_hosts", "example.com",
		"-ignore_hosts", "test.com",
		"-debug", "1",
		"-scan_pii",
		"-scan_tech",
		"-dns_retries", "5",
	}

	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if config.Addr != ":1234" {
		t.Errorf("Addr mismatch: %s", config.Addr)
	}
	if config.WebAddr != ":5678" {
		t.Errorf("WebAddr mismatch: %s", config.WebAddr)
	}
	if !config.SslInsecure {
		t.Error("SslInsecure should be true")
	}
	if len(config.IgnoreHosts) != 2 || config.IgnoreHosts[0] != "example.com" {
		t.Errorf("IgnoreHosts mismatch: %v", config.IgnoreHosts)
	}
	if config.Debug != 1 {
		t.Errorf("Debug mismatch: %d", config.Debug)
	}
	if !config.ScanPII {
		t.Error("ScanPII should be true")
	}
	if !config.ScanTech {
		t.Error("ScanTech should be true")
	}
	if config.DnsRetries != 5 {
		t.Errorf("DnsRetries mismatch: %d", config.DnsRetries)
	}
}

func TestLoadConfigFromCli(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"cmd", "-addr", ":1111"}
	config := loadConfigFromCli()
	if config.Addr != ":1111" {
		t.Errorf("Expected addr :1111, got %s", config.Addr)
	}
}

func TestLoadConfig_Version(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"cmd", "-version"}
	config := loadConfig()
	if !config.version {
		t.Error("Expected version true")
	}
}

func TestLoadConfig_File(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "cfg.json")
	os.WriteFile(path, []byte(`{"cert_path": "/tmp/certs"}`), 0644)
	
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"cmd", "-f", path, "-debug", "1"}
	config := loadConfig()
	if config.CertPath != "/tmp/certs" {
		t.Errorf("Expected CertPath /tmp/certs from file, got %s", config.CertPath)
	}
	if config.Debug != 1 {
		t.Error("Expected debug 1 from CLI override")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")
	os.WriteFile(path, []byte("{invalid}"), 0644)
	
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"cmd", "-f", path}
	config := loadConfig()
	if config.Addr != ":9080" {
		t.Errorf("Expected default addr, got %s", config.Addr)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"cmd", "-f", "non-existent-config.json"}
	config := loadConfig()
	if config.Addr != ":9080" {
		t.Errorf("Expected default addr, got %s", config.Addr)
	}
}
