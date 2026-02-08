package addon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewMapRemoteFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Valid JSON - assuming Object structure based on code
	validConfig := `{
		"Items": [
			{
				"From": {"Host": "example.com"},
				"To": {"Host": "test.com"},
				"Enable": true
			}
		],
		"Enable": true
	}`
	validFile := filepath.Join(tmpDir, "map_remote_valid.json")
	os.WriteFile(validFile, []byte(validConfig), 0644)

	mr, err := NewMapRemoteFromFile(validFile)
	if err != nil {
		t.Fatalf("NewMapRemoteFromFile valid failed: %v", err)
	}
	if len(mr.Items) != 1 {
		t.Errorf("Expected 1 config item, got %d", len(mr.Items))
	}

	// Invalid JSON
	invalidFile := filepath.Join(tmpDir, "map_remote_invalid.json")
	os.WriteFile(invalidFile, []byte(`{invalid`), 0644)
	
	_, err = NewMapRemoteFromFile(invalidFile)
	if err == nil {
		t.Error("NewMapRemoteFromFile invalid JSON expected error, got nil")
	}

	// Non-existent file
	_, err = NewMapRemoteFromFile(filepath.Join(tmpDir, "non_existent.json"))
	if err == nil {
		t.Error("NewMapRemoteFromFile non-existent expected error, got nil")
	}

	// Invalid Logic (MapFrom missing)
	invalidLogic := `{"Items": [{"To": {"Host": "test.com"}}], "Enable": true}`
	invalidLogicFile := filepath.Join(tmpDir, "map_remote_logic_invalid.json")
	os.WriteFile(invalidLogicFile, []byte(invalidLogic), 0644)
	_, err = NewMapRemoteFromFile(invalidLogicFile)
	if err == nil {
		t.Error("NewMapRemoteFromFile invalid logic expected error, got nil")
	}
}

func TestNewMapLocalFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid JSON
	validConfig := `{
		"Items": [
			{
				"From": {"Host": "example.com"},
				"To": {"Path": "/tmp/test"},
				"Enable": true
			}
		],
		"Enable": true
	}`
	validFile := filepath.Join(tmpDir, "map_local_valid.json")
	os.WriteFile(validFile, []byte(validConfig), 0644)

	ml, err := NewMapLocalFromFile(validFile)
	if err != nil {
		t.Fatalf("NewMapLocalFromFile valid failed: %v", err)
	}
	if len(ml.Items) != 1 {
		t.Errorf("Expected 1 config item, got %d", len(ml.Items))
	}

	// Invalid JSON
	invalidFile := filepath.Join(tmpDir, "map_local_invalid.json")
	os.WriteFile(invalidFile, []byte(`{invalid`), 0644)

	_, err = NewMapLocalFromFile(invalidFile)
	if err == nil {
		t.Error("NewMapLocalFromFile invalid JSON expected error, got nil")
	}
	
	// Invalid Logic
	invalidLogic := `{"Items": [{"From": {"Host": "example.com"}}], "Enable": true}` // Missing To
	invalidLogicFile := filepath.Join(tmpDir, "map_local_logic_invalid.json")
	os.WriteFile(invalidLogicFile, []byte(invalidLogic), 0644)
	_, err = NewMapLocalFromFile(invalidLogicFile)
	if err == nil {
		t.Error("NewMapLocalFromFile invalid logic expected error, got nil")
	}
}
