package credential

import (
	"context"
	"encoding/hex"
	"os"
	"testing"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/paularlott/cli"
)

func TestRotateKeyCommand(t *testing.T) {
	// Create a temporary directory for the database
	tmpDir, err := os.MkdirTemp("", "rackd-cred-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	newKeyHex := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

	oldKey, _ := hex.DecodeString(oldKeyHex)
	newKey, _ := hex.DecodeString(newKeyHex)

	// Step 1: Initialize database and seed a credential using the old key
	store, err := storage.NewExtendedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	oldCredStore, err := credentials.NewSQLiteStorage(store.DB(), oldKey)
	if err != nil {
		t.Fatalf("failed to initialize old cred store: %v", err)
	}

	testCred := &model.Credential{
		Name:          "Test Device",
		Type:          "snmp_v2c",
		DatacenterID:  "dc-1",
		SNMPCommunity: "public123",
	}

	if err := oldCredStore.Create(testCred); err != nil {
		t.Fatalf("failed to create test credential: %v", err)
	}
	store.Close() // Close the db so the CLI command can open it

	// Step 2: Set environment variable and run the command
	os.Setenv("ENCRYPTION_KEY", oldKeyHex)
	defer os.Unsetenv("ENCRYPTION_KEY")

	cmd := RotateKeyCommand()

	// Create a CLI App context
	app := &cli.Command{
		Name:     "test-app",
		Commands: []*cli.Command{cmd},
	}

	oldArgs := os.Args
	os.Args = []string{"rackd", "rotate-key", "--data-dir", tmpDir, "--new-key", newKeyHex}
	defer func() { os.Args = oldArgs }()

	// Execute the command directly
	err = app.Execute(context.Background())
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Step 3: Verify the credential was re-encrypted with the new key
	verifyStore, err := storage.NewExtendedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to re-open storage: %v", err)
	}
	defer verifyStore.Close()

	// 3a. Verify reading with old key fails decryption (returns error)
	oldVerifyStore, err := credentials.NewSQLiteStorage(verifyStore.DB(), oldKey)
	if err == nil {
		_, err = oldVerifyStore.Get(testCred.ID)
		if err == nil {
			t.Errorf("expected reading with old key to fail decryption, but it succeeded")
		}
	}

	// 3b. Verify reading with new key succeeds
	newVerifyStore, err := credentials.NewSQLiteStorage(verifyStore.DB(), newKey)
	if err != nil {
		t.Fatalf("failed to init verify cred store with new key: %v", err)
	}

	decryptedCred, err := newVerifyStore.Get(testCred.ID)
	if err != nil {
		t.Fatalf("failed to read credential with new key: %v", err)
	}

	if decryptedCred.SNMPCommunity != "public123" {
		t.Errorf("decrypted SNMPCommunity = %v, want %v", decryptedCred.SNMPCommunity, "public123")
	}
}
