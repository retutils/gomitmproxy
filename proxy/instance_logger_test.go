package proxy

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestInstanceLogger(t *testing.T) {
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetOutput(os.Stderr)

	logger := NewInstanceLogger(":8080", "test")
	
	// Test all wrapper methods
	logger.Info("info msg")
	if !strings.Contains(buf.String(), "info msg") {
		t.Error("Info failed")
	}
	buf.Reset()

	logger.Infof("infof %s", "msg")
	if !strings.Contains(buf.String(), "infof msg") {
		t.Error("Infof failed")
	}
	buf.Reset()

	logger.Debug("debug msg")
	if !strings.Contains(buf.String(), "debug msg") {
		t.Error("Debug failed")
	}
	buf.Reset()

	logger.Debugf("debugf %s", "msg")
	if !strings.Contains(buf.String(), "debugf msg") {
		t.Error("Debugf failed")
	}
	buf.Reset()

	logger.Warn("warn msg")
	if !strings.Contains(buf.String(), "warn msg") {
		t.Error("Warn failed")
	}
	buf.Reset()

	logger.Warnf("warnf %s", "msg")
	if !strings.Contains(buf.String(), "warnf msg") {
		t.Error("Warnf failed")
	}
	buf.Reset()

	logger.Error("error msg")
	if !strings.Contains(buf.String(), "error msg") {
		t.Error("Error failed")
	}
	buf.Reset()

	logger.Errorf("errorf %s", "msg")
	if !strings.Contains(buf.String(), "errorf msg") {
		t.Error("Errorf failed")
	}
	buf.Reset()
	
	// GetEntry
	entry := logger.GetEntry()
	if entry == nil {
		t.Error("GetEntry returned nil")
	}
	
	// Test file logger path (integrationish, or skip file creation)
	// NewInstanceLoggerWithFile with file path creates a file.
	tmpfile, err := os.CreateTemp("", "logtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	
	fileLogger := NewInstanceLoggerWithFile(":8081", "filetest", tmpfile.Name())
	fileLogger.Info("file msg")
	
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "file msg") {
		t.Error("File logger didn't write to file")
	}
}
