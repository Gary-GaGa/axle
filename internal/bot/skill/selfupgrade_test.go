package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBumpVersion(t *testing.T) {
	// Setup temp dir with a version.go
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "internal", "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `package app

// Version is the current Axle release version (SemVer).
const Version = "1.2.3"
`
	vf := filepath.Join(appDir, "version.go")
	if err := os.WriteFile(vf, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	newVer, err := BumpVersion(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if newVer != "1.2.4" {
		t.Errorf("expected 1.2.4, got %s", newVer)
	}

	// Verify file updated
	data, _ := os.ReadFile(vf)
	if !strings.Contains(string(data), `"1.2.4"`) {
		t.Errorf("file not updated: %s", string(data))
	}
}

func TestBumpVersionMissing(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := BumpVersion(tmpDir)
	if err == nil {
		t.Error("expected error for missing version.go")
	}
}

func TestUpgradeBackupRollback(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "axle")

	// Create a fake binary
	if err := os.WriteFile(binPath, []byte("binary-v1"), 0755); err != nil {
		t.Fatal(err)
	}

	// Backup
	if err := UpgradeBackupBinary(tmpDir); err != nil {
		t.Fatal(err)
	}

	bakPath := filepath.Join(tmpDir, "axle.bak")
	if _, err := os.Stat(bakPath); err != nil {
		t.Fatal("backup file should exist")
	}

	// Overwrite the binary
	if err := os.WriteFile(binPath, []byte("binary-v2"), 0755); err != nil {
		t.Fatal(err)
	}

	// Rollback
	if err := UpgradeRollbackBinary(tmpDir); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(binPath)
	if string(data) != "binary-v1" {
		t.Errorf("expected binary-v1 after rollback, got %s", string(data))
	}
}
