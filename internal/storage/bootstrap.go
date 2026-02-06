package storage

import (
	"fmt"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/log"
)

func BootstrapInitialAdmin(store ExtendedStorage, cfg *config.Config, sessionManager interface{}) error {
	userCount, err := store.UserCount()
	if err != nil {
		return fmt.Errorf("failed to check for existing users: %w", err)
	}

	if userCount > 0 {
		log.Info("Users already exist, skipping initial admin creation", "count", userCount)
		return nil
	}

	log.Info("No users found, checking for initial admin configuration")

	if cfg.InitialAdminUsername == "" || cfg.InitialAdminPassword == "" {
		log.Warn("No initial admin configured via environment variables")
		log.Warn("To create the first admin user, set the following environment variables:")
		log.Warn("  INITIAL_ADMIN_USERNAME - Username for the initial admin user")
		log.Warn("  INITIAL_ADMIN_PASSWORD - Password for the initial admin user (min 8 characters)")
		log.Warn("  INITIAL_ADMIN_EMAIL - Email for the initial admin user (optional, default: admin@localhost)")
		log.Warn("  INITIAL_ADMIN_FULL_NAME - Full name for the initial admin user (optional, default: 'System Administrator')")
		log.Warn("")
		log.Warn("Alternatively, create an admin user via CLI after starting the server:")
		log.Warn("  rackd user create --username admin --email admin@example.com")
		return nil
	}

	log.Info("Creating initial admin user", "username", cfg.InitialAdminUsername)

	if err := store.CreateInitialAdmin(
		cfg.InitialAdminUsername,
		cfg.InitialAdminEmail,
		cfg.InitialAdminFullName,
		cfg.InitialAdminPassword,
	); err != nil {
		return fmt.Errorf("failed to create initial admin user: %w", err)
	}

	log.Info("Initial admin user created successfully", "username", cfg.InitialAdminUsername, "email", cfg.InitialAdminEmail)
	log.Warn("⚠️  IMPORTANT: You should change the initial admin password after first login")

	return nil
}
