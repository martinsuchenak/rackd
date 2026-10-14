package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/sqlite"
	"github.com/martinsuchenak/xdal/pkg/xdal"

	"github.com/martinsuchenak/rackd/internal/log"
)

// XDAL returns the database wrapped for xdal: its query builders, schema
// builder and named locks.
func (s *SQLiteStorage) XDAL() xdal.DBInterface {
	return sqlite.NewSQLiteDB(s.db, nil)
}

// RunExtensionMigrations applies an extension's migrations in its own
// namespace of the migration tracking table. Like core's, they are backed up
// first when the extension already has migrations applied.
func (s *SQLiteStorage) RunExtensionMigrations(ctx context.Context, namespace string, steps []migrate.Migration) (MigrationResult, error) {
	return runNamespaceMigrations(ctx, s.db, namespace, steps, s.migrateOpts)
}

func runNamespaceMigrations(ctx context.Context, db *sql.DB, namespace string, steps []migrate.Migration, opts MigrationOptions) (MigrationResult, error) {
	var res MigrationResult
	if namespace == CoreMigrationNamespace || namespace == legacyAdoptionNamespace {
		return res, fmt.Errorf("migration namespace %q is reserved", namespace)
	}
	m, err := migrate.New(sqlite.NewSQLiteDB(db, nil), steps, migrate.Options{Namespace: namespace, Logger: migrationLogger{}})
	if err != nil {
		return res, fmt.Errorf("%s migrations: %w", namespace, err)
	}
	pending, err := m.Pending(ctx)
	if err != nil {
		return res, fmt.Errorf("%s migrations: %w", namespace, migrationError(err, ""))
	}
	if len(pending) == 0 {
		return res, nil
	}
	if len(pending) < len(steps) {
		res.BackupPath, err = preMigrationBackup(ctx, db, opts, "up-"+namespace+"-"+pending[len(pending)-1].Version)
		if err != nil {
			return res, err
		}
	}
	res.Versions, err = m.Up(ctx)
	if err != nil {
		return res, fmt.Errorf("%s migrations: %w", namespace, migrationError(err, res.BackupPath))
	}
	log.Info("Applied extension migrations", "extension", namespace, "count", len(res.Versions), "latest", res.Versions[len(res.Versions)-1], "backup", res.BackupPath)
	return res, nil
}

// PermissionGrant is a permission an extension declares, with the built-in
// roles that get it when it is first added.
type PermissionGrant struct {
	Resource     string
	Action       string
	DefaultRoles []string
}

// EnsurePermissions adds the permissions that don't exist yet and grants each
// new one to its default roles. Existing permissions and their grants are left
// alone, so changes administrators made are kept. RBAC is rebuilt when anything
// was added.
func (s *SQLiteStorage) EnsurePermissions(ctx context.Context, grants []PermissionGrant) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	added, err := ensurePermissions(ctx, tx, grants)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if added > 0 {
		log.Info("Added extension permissions", "count", added)
		return s.RebuildRBAC(ctx)
	}
	return nil
}

func ensurePermissions(ctx context.Context, tx *sql.Tx, grants []PermissionGrant) (int, error) {
	added := 0
	now := nowUTC()
	for _, g := range grants {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM permissions WHERE resource = ? AND action = ?)`,
			g.Resource, g.Action).Scan(&exists); err != nil {
			return added, err
		}
		if exists {
			continue
		}
		id := newUUID()
		if _, err := tx.ExecContext(ctx, `INSERT INTO permissions (id, name, resource, action, created_at) VALUES (?, ?, ?, ?, ?)`,
			id, g.Resource+":"+g.Action, g.Resource, g.Action, now); err != nil {
			return added, fmt.Errorf("failed to add permission %s:%s: %w", g.Resource, g.Action, err)
		}
		for _, role := range g.DefaultRoles {
			res, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO role_permissions (role_id, permission_id, created_at)
				SELECT id, ?, ? FROM roles WHERE name = ?`, id, now, role)
			if err != nil {
				return added, fmt.Errorf("failed to grant %s:%s to %s: %w", g.Resource, g.Action, role, err)
			}
			if n, _ := res.RowsAffected(); n == 0 {
				log.Warn("Default role for extension permission not found", "role", role, "permission", g.Resource+":"+g.Action)
			}
		}
		added++
	}
	return added, nil
}

// RecordAudit adds an entry to the audit log for the caller in ctx (see
// service.WithAuditContext). It does nothing when ctx carries no caller.
func (s *SQLiteStorage) RecordAudit(ctx context.Context, action, resource, resourceID string, changes any) {
	s.auditLog(ctx, action, resource, resourceID, changes)
}
