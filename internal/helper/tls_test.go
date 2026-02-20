package helper

import (
	"testing"
)

func TestGetTlsKeyLogWriter_Basic(t *testing.T) {
	// Just call it to ensure no panic and get coverage
	_ = GetTlsKeyLogWriter()
}
