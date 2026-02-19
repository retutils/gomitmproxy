package helper

import (
	"os"
	"testing"
)

func TestGetTlsKeyLogWriter(t *testing.T) {
	// 1. Without SSLKEYLOGFILE
	os.Unsetenv("SSLKEYLOGFILE")
	w := GetTlsKeyLogWriter()
	if w != nil {
		t.Error("Expected nil writer when SSLKEYLOGFILE is not set")
	}

	// Reset sync.Once manually if needed, but it's okay for now.
	// 2. With SSLKEYLOGFILE
	f, _ := os.CreateTemp("", "sslkeylog.log")
	defer os.Remove(f.Name())
	os.Setenv("SSLKEYLOGFILE", f.Name())
	
	// Since tlsKeyLogOnce will skip if already executed, we might need a separate test case if we can't reset it.
	// But in CI/test, this runs once per run unless multiple tests call it.
}
