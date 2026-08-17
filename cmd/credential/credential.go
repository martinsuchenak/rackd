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

			oldEncryptor, err := credentials.NewEncryptor(oldKey)
			if err != nil {
				return fmt.Errorf("failed to create old-key encryptor: %w", err)
			}
			newEncryptor, err := credentials.NewEncryptor(newKey)
			if err != nil {
				return fmt.Errorf("failed to create new-key encryptor: %w", err)
			}

			// Re-encrypt DNS provider tokens (encrypted with the same master key)
			if err := rotateColumn(store, oldEncryptor, newEncryptor,
				"Re-encrypting DNS provider tokens...",
				"SELECT id, token FROM dns_provider_configs WHERE token != ''",
				"UPDATE dns_provider_configs SET token = ? WHERE id = ?"); err != nil {
				return err
			}

			// Re-encrypt webhook signing secrets
			if err := rotateColumn(store, oldEncryptor, newEncryptor,
				"Re-encrypting webhook secrets...",
				"SELECT id, secret FROM webhooks WHERE secret != ''",
				"UPDATE webhooks SET secret = ? WHERE id = ?"); err != nil {
				return err
			}

			fmt.Println("Successfully rotated encryption key! Please update your ENCRYPTION_KEY environment variable with the new key.")
			return nil
		},
	}
}

// rotateColumn decrypts a column with the old key and re-encrypts it with the
// new key. Rows that fail decryption (legacy plaintext or empty) are skipped.
func rotateColumn(store storage.ExtendedStorage, oldEnc, newEnc *credentials.Encryptor, header, selectSQL, updateSQL string) error {
	rows, err := store.DB().Query(selectSQL)
	if err != nil {
		return fmt.Errorf("failed to read rows for rotation: %w", err)
	}
	defer rows.Close()

	type pending struct {
		id    string
		reEnc string
	}
	var updates []pending
	for rows.Next() {
		var id, value string
		if err := rows.Scan(&id, &value); err != nil {
			return fmt.Errorf("failed to scan row for rotation: %w", err)
		}
		plaintext, err := oldEnc.Decrypt(value)
		if err != nil {
			// Not encrypted with the old key (legacy plaintext) — skip.
			continue
		}
		reEncrypted, err := newEnc.Encrypt(plaintext)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt row %s: %w", id, err)
		}
		updates = append(updates, pending{id: id, reEnc: reEncrypted})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed iterating rows for rotation: %w", err)
	}

	fmt.Printf("%s (%d rows)\n", header, len(updates))
	for _, u := range updates {
		if _, err := store.DB().Exec(updateSQL, u.reEnc, u.id); err != nil {
			return fmt.Errorf("failed to persist re-encrypted row %s: %w", u.id, err)
		}
	}
	return nil
}
