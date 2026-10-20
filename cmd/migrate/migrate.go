package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/cmd/internal/flags"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/paularlott/cli"

	_ "modernc.org/sqlite"
)

// Command returns the migrate command for core's migrations only.
func Command() *cli.Command {
	return NewCommand(nil)
}

// NewCommand returns the migrate command, covering core's migrations and those
// of the features compiled into the binary.
func NewCommand(features []extension.Feature) *cli.Command {
	exts := extensionMigrations(features)
	return &cli.Command{
		Name:  "migrate",
		Usage: "Database migration management",
		Commands: []*cli.Command{
			statusCommand(exts),
			runCommand(exts),
			rollbackCommand(exts),
		},
	}
}

// extensionSteps are one feature's migrations.
type extensionSteps struct {
	name  string
	steps []extension.Migration
}

func extensionMigrations(features []extension.Feature) []extensionSteps {
	var out []extensionSteps
	for _, f := range features {
		if mp, ok := f.(extension.MigrationProvider); ok {
			out = append(out, extensionSteps{name: f.Manifest().Name, steps: mp.Migrations()})
		}
	}
	return out
}

func findExtension(exts []extensionSteps, name string) (extensionSteps, bool) {
	for _, e := range exts {
		if e.name == name {
			return e, true
		}
	}
	return extensionSteps{}, false
}

func namespaceFlag() cli.Flag {
	return &cli.StringFlag{Name: "namespace", Usage: "Migrations to act on: \"core\" or an extension's name", DefaultValue: storage.CoreMigrationNamespace}
}

func dataDirFlag() cli.Flag { return flags.DataDir() }

func resolveDataDir(cmd *cli.Command) string { return flags.ResolveDataDir(cmd) }

func statusCommand(exts []extensionSteps) *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show migration status for core and each extension",
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
			pending := printStatus(storage.CoreMigrationNamespace, statuses, false)

			inBinary := map[string]bool{}
			for _, e := range exts {
				inBinary[e.name] = true
				statuses, err := storage.ExtensionMigrationStatus(ctx, db, e.name, e.steps)
				if err != nil {
					return fmt.Errorf("failed to get %s migration status: %w", e.name, err)
				}
				pending += printStatus(e.name, statuses, false)
			}

			recorded, err := storage.RecordedExtensionNamespaces(ctx, db)
			if err != nil {
				return fmt.Errorf("failed to list extension migrations: %w", err)
			}
			for _, ns := range recorded {
				if inBinary[ns] {
					continue
				}
				statuses, err := storage.ExtensionMigrationStatus(ctx, db, ns, nil)
				if err != nil {
					return fmt.Errorf("failed to get %s migration status: %w", ns, err)
				}
				fmt.Printf("%s (extension not in this binary; its tables are kept)\n", ns)
				printStatus("", statuses, true)
			}

			fmt.Printf("Total: %d pending\n", pending)
			return nil
		},
	}
}

// printStatus prints a namespace's migrations and returns how many are pending.
// For an extension not in this binary every recorded migration is "unknown" to
// it; those are shown as applied.
func printStatus(namespace string, statuses []storage.MigrationStatus, foreign bool) int {
	if namespace != "" {
		fmt.Println(namespace)
	}
	pending := 0
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "  VERSION\tNAME\tSTATUS\tAPPLIED AT")
	for _, s := range statuses {
		status := "applied"
		appliedAt := s.AppliedAt
		switch {
		case s.Dirty:
			status = "dirty"
		case s.Unknown && foreign:
			status = "applied"
		case s.Unknown:
			status = "unknown (newer or removed)"
		case !s.Applied:
			status = "pending"
			appliedAt = "-"
			pending++
		}
		_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", s.Version, s.Name, status, appliedAt)
	}
	_ = w.Flush()
	fmt.Println()
	return pending
}

func runCommand(exts []extensionSteps) *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Run pending migrations for core and each extension, backing up the database first",
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
			applied := 0
			res, err := storage.RunMigrationsWithOptions(ctx, db, opts)
			if err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}
			applied += report(storage.CoreMigrationNamespace, res)
			if res.BackupPath != "" {
				opts.BackupDir = "" // one backup covers this whole run
			}
			for _, e := range exts {
				res, err := storage.RunExtensionMigrationsOn(ctx, db, e.name, e.steps, opts)
				if err != nil {
					return fmt.Errorf("%s migration failed: %w", e.name, err)
				}
				applied += report(e.name, res)
				if res.BackupPath != "" {
					opts.BackupDir = ""
				}
			}
			if applied == 0 {
				fmt.Println("Database is up to date, no pending migrations")
			}
			return nil
		},
	}
}

func report(namespace string, res storage.MigrationResult) int {
	if res.BackupPath != "" {
		fmt.Printf("Backup created: %s\n", res.BackupPath)
	}
	if len(res.Versions) > 0 {
		fmt.Printf("%s: applied %d migration(s)\n", namespace, len(res.Versions))
	}
	return len(res.Versions)
}

func rollbackCommand(exts []extensionSteps) *cli.Command {
	return &cli.Command{
		Name:  "rollback",
		Usage: "Revert applied migrations of core or one extension, backing up the database first",
		Flags: []cli.Flag{
			dataDirFlag(),
			namespaceFlag(),
			&cli.IntFlag{Name: "steps", Usage: "Number of migrations to revert", DefaultValue: 1},
			&cli.StringFlag{Name: "to", Usage: "Revert every migration newer than this version"},
			&cli.BoolFlag{Name: "yes", Usage: "Confirm the rollback; reverting can drop tables and data"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			namespace := cmd.GetString("namespace")
			steps := cmd.GetInt("steps")
			to := cmd.GetString("to")
			if steps < 1 && to == "" {
				return fmt.Errorf("--steps must be at least 1")
			}
			var ext extensionSteps
			if namespace != storage.CoreMigrationNamespace {
				var ok bool
				if ext, ok = findExtension(exts, namespace); !ok {
					return fmt.Errorf("extension %q is not in this binary, so its migrations can't be reverted with it", namespace)
				}
			}
			target := fmt.Sprintf("the latest %d %s migration(s)", steps, namespace)
			if to != "" {
				target = fmt.Sprintf("every %s migration newer than %s", namespace, to)
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

			opts := storage.MigrationOptions{BackupDir: filepath.Join(dataDir, "backups")}
			var res storage.MigrationResult
			if namespace == storage.CoreMigrationNamespace {
				res, err = storage.RollbackMigrations(ctx, db, steps, to, opts)
			} else {
				res, err = storage.RollbackExtensionMigrations(ctx, db, namespace, ext.steps, steps, to, opts)
			}
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
				fmt.Printf("Reverted %s %s\n", namespace, v)
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
