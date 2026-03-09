package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/paularlott/cli"

	_ "modernc.org/sqlite"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Database migration management",
		Commands: []*cli.Command{
			statusCommand(),
			runCommand(),
		},
	}
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show migration status",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "data-dir", Usage: "Data directory", DefaultValue: "./data"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config.Load()
			dataDir := cmd.GetString("data-dir")
			if dataDir == "" {
				dataDir = cfg.DataDir
			}

			db, err := openDB(dataDir)
			if err != nil {
				return err
			}
			defer db.Close()

			statuses, err := storage.GetMigrationStatus(ctx, db)
			if err != nil {
				return fmt.Errorf("failed to get migration status: %w", err)
			}

			pending := 0
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "VERSION\tNAME\tSTATUS\tAPPLIED AT")
			for _, s := range statuses {
				status := "applied"
				appliedAt := s.AppliedAt
				if !s.Applied {
					status = "pending"
					appliedAt = "-"
					pending++
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Version, s.Name, status, appliedAt)
			}
			w.Flush()

			fmt.Printf("\nTotal: %d migrations, %d pending\n", len(statuses), pending)
			return nil
		},
	}
}

func runCommand() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Run pending migrations",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "data-dir", Usage: "Data directory", DefaultValue: "./data"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config.Load()
			dataDir := cmd.GetString("data-dir")
			if dataDir == "" {
				dataDir = cfg.DataDir
			}

			db, err := openDB(dataDir)
			if err != nil {
				return err
			}
			defer db.Close()

			// Check pending count first
			statuses, err := storage.GetMigrationStatus(ctx, db)
			if err != nil {
				return fmt.Errorf("failed to get migration status: %w", err)
			}

			pending := 0
			for _, s := range statuses {
				if !s.Applied {
					pending++
				}
			}

			if pending == 0 {
				fmt.Println("Database is up to date, no pending migrations")
				return nil
			}

			fmt.Printf("Running %d pending migration(s)...\n", pending)
			if err := storage.RunMigrations(ctx, db); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			fmt.Println("All migrations applied successfully")
			return nil
		},
	}
}

func openDB(dataDir string) (*sql.DB, error) {
	dbPath := filepath.Join(dataDir, "rackd.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found at %s", dbPath)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}
