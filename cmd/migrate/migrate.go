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
			rollbackCommand(),
		},
	}
}

func dataDirFlag() cli.Flag {
	return &cli.StringFlag{Name: "data-dir", Usage: "Data directory", DefaultValue: "./data"}
}

func resolveDataDir(cmd *cli.Command) string {
	if dataDir := cmd.GetString("data-dir"); dataDir != "" {
		return dataDir
	}
	return config.Load().DataDir
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show migration status",
		Flags: []cli.Flag{dataDirFlag()},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			db, err := openDB(resolveDataDir(cmd))
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()

			statuses, err := storage.GetMigrationStatus(ctx, db)
			if err != nil {
				return fmt.Errorf("failed to get migration status: %w", err)
			}

			pending := 0
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "VERSION\tNAME\tSTATUS\tAPPLIED AT")
			for _, s := range statuses {
				status := "applied"
				appliedAt := s.AppliedAt
				switch {
				case s.Dirty:
					status = "dirty"
				case s.Unknown:
					status = "unknown (newer rackd)"
				case !s.Applied:
					status = "pending"
					appliedAt = "-"
					pending++
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Version, s.Name, status, appliedAt)
			}
			_ = w.Flush()

			fmt.Printf("\nTotal: %d migrations, %d pending\n", len(statuses), pending)
			return nil
		},
	}
}

func runCommand() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Run pending migrations, backing up the database first",
		Flags: []cli.Flag{
			dataDirFlag(),
			&cli.BoolFlag{Name: "no-backup", Usage: "Skip the pre-migration backup"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			dataDir := resolveDataDir(cmd)
			db, err := openDB(dataDir)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()

			opts := storage.MigrationOptions{BackupDir: filepath.Join(dataDir, "backups")}
			if cmd.GetBool("no-backup") {
				opts.BackupDir = ""
			}
			res, err := storage.RunMigrationsWithOptions(ctx, db, opts)
			if err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}
			if len(res.Versions) == 0 {
				fmt.Println("Database is up to date, no pending migrations")
				return nil
			}
			if res.BackupPath != "" {
				fmt.Printf("Backup created: %s\n", res.BackupPath)
			}
			fmt.Printf("Applied %d migration(s)\n", len(res.Versions))
			return nil
		},
	}
}

func rollbackCommand() *cli.Command {
	return &cli.Command{
		Name:  "rollback",
		Usage: "Revert applied migrations, backing up the database first",
		Flags: []cli.Flag{
			dataDirFlag(),
			&cli.IntFlag{Name: "steps", Usage: "Number of migrations to revert", DefaultValue: 1},
			&cli.StringFlag{Name: "to", Usage: "Revert every migration newer than this version"},
			&cli.BoolFlag{Name: "yes", Usage: "Confirm the rollback; reverting can drop tables and data"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			steps := cmd.GetInt("steps")
			to := cmd.GetString("to")
			if steps < 1 && to == "" {
				return fmt.Errorf("--steps must be at least 1")
			}
			target := fmt.Sprintf("the latest %d migration(s)", steps)
			if to != "" {
				target = "every migration newer than " + to
			}
			if !cmd.GetBool("yes") {
				fmt.Printf("This reverts %s and can drop tables and data.\nA backup is taken first. Re-run with --yes to continue.\n", target)
				return nil
			}

			dataDir := resolveDataDir(cmd)
			db, err := openDB(dataDir)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()

			res, err := storage.RollbackMigrations(ctx, db, steps, to, storage.MigrationOptions{BackupDir: filepath.Join(dataDir, "backups")})
			if err != nil {
				return fmt.Errorf("rollback failed: %w", err)
			}
			if res.BackupPath != "" {
				fmt.Printf("Backup created: %s\n", res.BackupPath)
			}
			if len(res.Versions) == 0 {
				fmt.Println("Nothing to roll back")
				return nil
			}
			for _, v := range res.Versions {
				fmt.Printf("Reverted %s\n", v)
			}
			return nil
		},
	}
}

func openDB(dataDir string) (*sql.DB, error) {
	dbPath := filepath.Join(dataDir, "rackd.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found at %s", dbPath)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}
