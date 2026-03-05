package credential

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/paularlott/cli"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "credentials",
		Usage: "Manage credentials and encryption",
		Commands: []*cli.Command{
			RotateKeyCommand(),
		},
	}
}

func RotateKeyCommand() *cli.Command {
	return &cli.Command{
		Name:  "rotate-key",
		Usage: "Rotate the master encryption key used for credentials",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "data-dir", Usage: "Data directory containing rackd.db", DefaultValue: "./data"},
			&cli.StringFlag{Name: "new-key", Usage: "New 32-byte hex-encoded encryption key", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			dataDir := cmd.GetString("data-dir")
			newKeyHex := cmd.GetString("new-key")

			oldKeyHex := os.Getenv("ENCRYPTION_KEY")
			if oldKeyHex == "" {
				return fmt.Errorf("ENCRYPTION_KEY environment variable is required to read existing credentials")
			}

			oldKey, err := hex.DecodeString(oldKeyHex)
			if err != nil {
				return fmt.Errorf("invalid existing ENCRYPTION_KEY (must be hex-encoded): %w", err)
			}
			if len(oldKey) != 32 {
				return fmt.Errorf("invalid existing ENCRYPTION_KEY (must be 32 bytes / 64 hex chars)")
			}

			newKey, err := hex.DecodeString(newKeyHex)
			if err != nil {
				return fmt.Errorf("invalid new-key (must be hex-encoded): %w", err)
			}
			if len(newKey) != 32 {
				return fmt.Errorf("invalid new-key (must be 32 bytes / 64 hex chars)")
			}

			// Open database connection
			store, err := storage.NewExtendedStorage(dataDir)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer store.Close()

			// Initialize credential storage with old key to read current encrypted rows
			oldCredStore, err := credentials.NewSQLiteStorage(store.DB(), oldKey)
			if err != nil {
				return fmt.Errorf("failed to initialize credential storage with old key: %w", err)
			}

			// Initialize credential storage with new key to write new encrypted rows
			newCredStore, err := credentials.NewSQLiteStorage(store.DB(), newKey)
			if err != nil {
				return fmt.Errorf("failed to initialize credential storage with new key: %w", err)
			}

			// Fetch all credentials (using empty datacenterID filters for all)
			creds, err := oldCredStore.List("")
			if err != nil {
				return fmt.Errorf("failed to list credentials with old key: %w", err)
			}

			fmt.Printf("Re-encrypting %d credentials...\n", len(creds))
			for i, cred := range creds {
				// newCredStore.Update writes back the credential encrypted with newKey
				if err := newCredStore.Update(&cred); err != nil {
					return fmt.Errorf("failed to re-encrypt credential %s (%s): %w", cred.ID, cred.Name, err)
				}
				fmt.Printf("[%d/%d] Rotated %s\n", i+1, len(creds), cred.Name)
			}

			fmt.Println("Successfully rotated encryption key! Please update your ENCRYPTION_KEY environment variable with the new key.")
			return nil
		},
	}
}
