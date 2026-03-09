package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "migrate" {
		t.Errorf("expected command name 'migrate', got %q", cmd.Name)
	}

	if len(cmd.Commands) != 2 {
		t.Errorf("expected 2 subcommands, got %d", len(cmd.Commands))
	}

	expectedSubcommands := []string{"status", "run"}
	for i, expected := range expectedSubcommands {
		if cmd.Commands[i].Name != expected {
			t.Errorf("subcommand %d: expected %q, got %q", i, expected, cmd.Commands[i].Name)
		}
	}
}

func TestSubcommandRunFunctions(t *testing.T) {
	cmd := Command()
	for _, sub := range cmd.Commands {
		if sub.Run == nil {
			t.Errorf("subcommand %q has nil Run function", sub.Name)
		}
	}
}

func TestOpenDB(t *testing.T) {
	// Test with non-existent path
	_, err := openDB("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent database, got nil")
	}
}

func TestOpenDBWithRealFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "rackd.db")

	// Create an empty file to simulate a DB
	f, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	f.Close()

	db, err := openDB(tmpDir)
	if err != nil {
		t.Fatalf("openDB failed: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}
