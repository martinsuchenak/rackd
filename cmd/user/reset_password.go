package user

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/paularlott/cli"
	"golang.org/x/term"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
)

// ResetPasswordCommand returns a command that resets a user's password
// directly against the database, without a running server or an
// authenticated API token. Intended as the recovery path when the only
// admin's password is lost. The server should be stopped while this runs.
func ResetPasswordCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset-password",
		Usage: "Reset a user's password directly in the database (no server required)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "data-dir", Usage: "Data directory containing rackd.db", DefaultValue: "./data"},
			&cli.StringFlag{Name: "username", Usage: "Username of the account to reset (e.g. admin)", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			dataDir := cmd.GetString("data-dir")
			username := cmd.GetString("username")

			// Read the new password from the terminal (hidden), matching the
			// bootstrap/API policy of at least 8 characters.
			fmt.Printf("Enter new password for %s: ", username)
			pw1, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			fmt.Println()

			fmt.Print("Confirm new password: ")
			pw2, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			fmt.Println()

			if string(pw1) != string(pw2) {
				return fmt.Errorf("passwords do not match")
			}
			if len(pw1) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}
			if strings.TrimSpace(string(pw1)) != string(pw1) {
				return fmt.Errorf("password must not start or end with whitespace")
			}

			// Open storage read/write. Logging is initialized to discard so
			// this command stays quiet on stdout.
			log.Init("console", "error", os.Stderr)

			store, err := storage.NewExtendedStorage(dataDir)
			if err != nil {
				return fmt.Errorf("failed to open database in %s: %w", dataDir, err)
			}
			defer store.Close()

			user, err := store.GetUserByUsername(ctx, username)
			if err != nil {
				return fmt.Errorf("user %q not found: %w", username, err)
			}

			hash, err := auth.HashPassword(string(pw1))
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}

			if err := store.UpdateUserPassword(ctx, user.ID, hash); err != nil {
				return fmt.Errorf("failed to update password: %w", err)
			}

			// Sessions are stored hashed, so existing sessions cannot be
			// looked up and invalidated by token here; remove any lingering
			// sessions for this user id directly.
			sessionStore := storage.NewSQLiteSessionStore(store.DB())
			sessionStore.DeleteByUser(ctx, user.ID)

			fmt.Printf("Password for %q reset successfully.\n", username)
			fmt.Println("All active sessions for this user have been invalidated.")
			fmt.Println("If the rackd server is currently running, restart it or have the user log in again with the new password.")
			return nil
		},
	}
}
