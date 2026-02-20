package storage

import (
	"testing"
	"time"
)

func TestService_HostTechnologies(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(tmpDir)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	defer svc.Close()

	hostname := "example.com"
	techs := []HostTechnology{
		{
			Hostname:     hostname,
			TechName:     "Nginx",
			Version:      "1.18.0",
			Categories:   "Web Server",
			LastDetected: time.Now(),
		},
		{
			Hostname:     hostname,
			TechName:     "React",
			Version:      "",
			Categories:   "Frontend Framework",
			LastDetected: time.Now(),
		},
	}

	// 1. Test Save (UPSERT)
	if err := svc.SaveHostTechnologies(hostname, techs); err != nil {
		t.Fatalf("SaveHostTechnologies failed: %v", err)
	}

	// 2. Test Get
	results, err := svc.GetHostTechnologies(hostname)
	if err != nil {
		t.Fatalf("GetHostTechnologies failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 technologies, got %d", len(results))
	}

	// 3. Test Update (UPSERT logic)
	updatedTechs := []HostTechnology{
		{
			Hostname:     hostname,
			TechName:     "Nginx",
			Version:      "1.20.0", // Updated version
			Categories:   "Web Server",
			LastDetected: time.Now(),
		},
	}
	if err := svc.SaveHostTechnologies(hostname, updatedTechs); err != nil {
		t.Fatalf("Update SaveHostTechnologies failed: %v", err)
	}

	results, err = svc.GetHostTechnologies(hostname)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 technologies after update, got %d", len(results))
	}

	foundNginx := false
	for _, tr := range results {
		if tr.TechName == "Nginx" {
			foundNginx = true
			if tr.Version != "1.20.0" {
				t.Errorf("Expected Nginx version 1.20.0, got %s", tr.Version)
			}
		}
	}
	if !foundNginx {
		t.Error("Nginx not found in results")
	}
}

func TestService_HostTechnologies_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	svc, _ := NewService(tmpDir)
	
	hostname := "error.com"
	techs := []HostTechnology{{TechName: "Test"}}

	// Close DB to trigger errors
	svc.db.Close()

	// 1. Save error
	if err := svc.SaveHostTechnologies(hostname, techs); err == nil {
		t.Error("Expected error from SaveHostTechnologies on closed DB")
	}

	// 2. Get error
	if _, err := svc.GetHostTechnologies(hostname); err == nil {
		t.Error("Expected error from GetHostTechnologies on closed DB")
	}
}
