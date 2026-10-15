package credential

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

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

// encryptedColumns are every column encrypted with the master key. Only
// webhook secrets may hold values that don't decrypt: Rackd stores them
// unencrypted when it runs without a key.
var encryptedColumns = []struct {
	table, column  string
	allowPlaintext bool
}{
	{"credentials", "snmp_community", false},
	{"credentials", "snmp_v3_user", false},
	{"credentials", "snmp_v3_auth", false},
	{"credentials", "snmp_v3_priv", false},
	{"credentials", "ssh_key_id", false},
	{"dns_provider_configs", "token", false},
	{"webhooks", "secret", true},
}

func RotateKeyCommand() *cli.Command {
	return &cli.Command{
		Name:  "rotate-key",
		Usage: "Re-encrypt credentials, DNS provider tokens and webhook secrets with a new key",
		Description: "Reads the current key from ENCRYPTION_KEY and the new one from the environment variable --new-key-env names " +
			"(NEW_ENCRYPTION_KEY by default) or from --new-key-file, so neither appears in the process list or shell history. " +
			"Stop the server first: everything is re-encrypted in one transaction, and a running server would keep writing with the old key.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "data-dir", Usage: "Data directory containing rackd.db", DefaultValue: "./data"},
			&cli.StringFlag{Name: "new-key-env", Usage: "Environment variable holding the new 64-hex-character key", DefaultValue: "NEW_ENCRYPTION_KEY"},
			&cli.StringFlag{Name: "new-key-file", Usage: "File holding the new key (overrides --new-key-env)"},
			&cli.BoolFlag{Name: "yes", Usage: "Confirm that the server is stopped"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			if !cmd.GetBool("yes") {
				return fmt.Errorf("stop the Rackd server, then run again with --yes")
			}
			oldKey, err := parseKey("ENCRYPTION_KEY", os.Getenv("ENCRYPTION_KEY"))
			if err != nil {
				return err
			}
			newKey, err := readNewKey(cmd.GetString("new-key-file"), cmd.GetString("new-key-env"))
			if err != nil {
				return err
			}

			store, err := storage.NewExtendedStorage(cmd.GetString("data-dir"))
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = store.Close() }()

			oldEnc, err := credentials.NewEncryptor(oldKey)
			if err != nil {
				return err
			}
			newEnc, err := credentials.NewEncryptor(newKey)
			if err != nil {
				return err
			}
			if err := rotateAll(ctx, store.DB(), oldEnc, newEnc); err != nil {
				return err
			}
			fmt.Println("Rotated the encryption key. Set ENCRYPTION_KEY to the new key before starting the server.")
			return nil
		},
	}
}

func parseKey(name, keyHex string) ([]byte, error) {
	if keyHex == "" {
		return nil, fmt.Errorf("%s is not set", name)
	}
	key, err := hex.DecodeString(strings.TrimSpace(keyHex))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("%s must be 64 hex characters (32 bytes)", name)
	}
	return key, nil
}

func readNewKey(file, envName string) ([]byte, error) {
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("reading the new key: %w", err)
		}
		return parseKey("the key in "+file, string(data))
	}
	return parseKey(envName, os.Getenv(envName))
}

// rotateAll re-encrypts every encrypted column in one transaction, so a
// failure leaves everything under the old key. A credential or DNS token that
// doesn't decrypt with the old key means a wrong ENCRYPTION_KEY, and nothing is
// written. A webhook secret that doesn't decrypt is left as it is: it may be
// stored unencrypted, or encrypted with another key, and encrypting it again
// would corrupt it in the second case.
func rotateAll(ctx context.Context, db *sql.DB, oldEnc, newEnc *credentials.Encryptor) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, c := range encryptedColumns {
		rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT id, %s FROM %s WHERE %s IS NOT NULL AND %s != ''", c.column, c.table, c.column, c.column))
		if err != nil {
			return fmt.Errorf("reading %s.%s: %w", c.table, c.column, err)
		}
		type update struct{ id, value string }
		var updates []update
		skipped := 0
		for rows.Next() {
			var id, value string
			if err := rows.Scan(&id, &value); err != nil {
				_ = rows.Close()
				return err
			}
			clear, err := oldEnc.Decrypt(value)
			if err != nil {
				if !c.allowPlaintext {
					_ = rows.Close()
					return fmt.Errorf("%s.%s of %s doesn't decrypt with ENCRYPTION_KEY; is it the current key? Nothing was changed", c.table, c.column, id)
				}
				skipped++
				continue
			}
			encrypted, err := newEnc.Encrypt(clear)
			if err != nil {
				_ = rows.Close()
				return err
			}
			updates = append(updates, update{id, encrypted})
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, u := range updates {
			if _, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", c.table, c.column), u.value, u.id); err != nil {
				return fmt.Errorf("writing %s.%s of %s: %w", c.table, c.column, u.id, err)
			}
		}
		fmt.Printf("%s.%s: %d re-encrypted", c.table, c.column, len(updates))
		if skipped > 0 {
			fmt.Printf("; %d left unchanged because they don't decrypt with ENCRYPTION_KEY (stored unencrypted, or under another key)", skipped)
		}
		fmt.Println()
	}
	return tx.Commit()
}
