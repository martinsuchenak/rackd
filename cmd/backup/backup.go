package backup

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/paularlott/cli"

	_ "modernc.org/sqlite"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "backup",
		Usage: "Back up the rackd database (safe while the server is running)",
		Flags: []cli.Flag{
			dataDirFlag(),
			&cli.StringFlag{Name: "output", Usage: "Output file path (default: rackd-backup-<timestamp>.db)"},
			&cli.BoolFlag{Name: "overwrite", Usage: "Replace the output file if it exists (only after the new backup is verified)"},
		},
		Commands: []*cli.Command{
			verifyCommand(),
			restoreCommand(),
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			dbPath := filepath.Join(resolveDataDir(cmd), "rackd.db")
			dst := cmd.GetString("output")
			if dst == "" {
				dst = fmt.Sprintf("rackd-backup-%s.db", time.Now().UTC().Format("20060102-150405"))
			}
			backup := storage.BackupFile
			if cmd.GetBool("overwrite") {
				backup = storage.BackupFileOverwrite
			}
			info, err := backup(ctx, dbPath, dst)
			if err != nil {
				return fmt.Errorf("backup failed: %w", err)
			}
			fmt.Printf("Backup created: %s (%.1f MB, %d migrations, verified)\n", dst, float64(info.SizeBytes)/1024/1024, info.Migrations)
			return nil
		},
	}
}

func dataDirFlag() cli.Flag {
	return &cli.StringFlag{Name: "data-dir", Usage: "Data directory (default: from config)", DefaultValue: "./data"}
}

func resolveDataDir(cmd *cli.Command) string {
	if dataDir := cmd.GetString("data-dir"); dataDir != "" {
		return dataDir
	}
	return config.Load().DataDir
}

func verifyCommand() *cli.Command {
	return &cli.Command{
		Name:  "verify",
		Usage: "Check a backup's integrity and migration history",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Usage: "Backup file to verify", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			info, err := storage.VerifyBackup(ctx, cmd.GetString("file"))
			if err != nil {
				return err
			}
			fmt.Printf("Backup OK: %s\n", info.Path)
			fmt.Printf("  Size:             %.1f MB\n", float64(info.SizeBytes)/1024/1024)
			fmt.Printf("  Migrations:       %d (latest %s)\n", info.Migrations, info.LatestMigration)
			if len(info.UnknownMigrations) > 0 {
				fmt.Printf("  Newer migrations: %s (made by a newer rackd; upgrade before restoring)\n", strings.Join(info.UnknownMigrations, ", "))
			}
			return nil
		},
	}
}

func restoreCommand() *cli.Command {
	return &cli.Command{
		Name:  "restore",
		Usage: "Replace the database with a backup (stop the server first)",
		Flags: []cli.Flag{
			dataDirFlag(),
			&cli.StringFlag{Name: "file", Usage: "Backup file to restore", Required: true},
			&cli.BoolFlag{Name: "yes", Usage: "Confirm replacing the current database"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			dbPath := filepath.Join(resolveDataDir(cmd), "rackd.db")
			file := cmd.GetString("file")
			if !cmd.GetBool("yes") {
				fmt.Printf("This replaces %s with %s.\nThe current database is kept alongside it. Re-run with --yes to continue.\n", dbPath, file)
				return nil
			}
			movedTo, err := storage.RestoreDatabase(ctx, file, dbPath)
			if err != nil {
				return fmt.Errorf("restore failed: %w", err)
			}
			fmt.Printf("Restored %s from %s\n", dbPath, file)
			if movedTo != "" {
				fmt.Printf("Previous database saved as %s\n", movedTo)
			}
			fmt.Println("Any pending migrations run on the next server start.")
			return nil
		},
	}
}
