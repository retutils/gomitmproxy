package main

import (
	"os"
	"testing"
)

func TestRun_Success(t *testing.T) {
	config := &Config{commonName: "example.com"}
	err := Run(config)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestRun_Error(t *testing.T) {
	config := &Config{commonName: ""}
	err := Run(config)
	if err == nil {
		t.Error("Expected error for empty commonName")
	}
}

func TestLoadConfig(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"cmd", "-commonName", "test.com"}
	config := loadConfig()
	if config.commonName != "test.com" {
		t.Errorf("Expected commonName test.com, got %s", config.commonName)
	}
}
