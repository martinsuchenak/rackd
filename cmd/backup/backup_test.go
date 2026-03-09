package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "backup" {
		t.Errorf("expected command name 'backup', got %q", cmd.Name)
	}

	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}

	if len(cmd.Flags) < 2 {
		t.Errorf("expected at least 2 flags (data-dir, output), got %d", len(cmd.Flags))
	}
}

func TestCopyFile(t *testing.T) {
	// Create a temp source file
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("test backup content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	dstPath := filepath.Join(tmpDir, "dest.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("expected %q, got %q", string(content), string(got))
	}
}

func TestCopyFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	err := copyFile(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest"))
	if err == nil {
		t.Error("expected error for non-existent source, got nil")
	}
}
