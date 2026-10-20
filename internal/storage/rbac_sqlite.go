package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	rbac "github.com/martinsuchenak/go-rbac"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type RBACStorage interface {
	CreatePermission(ctx context.Context, perm *model.Permission) error
	GetPermission(ctx context.Context, id string) (*model.Permission, error)
	GetPermissionByName(ctx context.Context, name string) (*model.Permission, error)
	ListPermissions(ctx context.Context, filter *model.PermissionFilter) ([]model.Permission, error)
	DeletePermission(ctx context.Context, id string) error

	CreateRole(ctx context.Context, role *model.Role) error
	GetRole(ctx context.Context, id string) (*model.Role, error)
	GetRoleByName(ctx context.Context, name string) (*model.Role, error)
	ListRoles(ctx context.Context, filter *model.RoleFilter) ([]model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) error
	DeleteRole(ctx context.Context, id string) error

	GetRolePermissions(ctx context.Context, roleID string) ([]model.Permission, error)
	SetRolePermissions(ctx context.Context, roleID string, permissionIDs []string) error
	AddRolePermission(ctx context.Context, roleID, permissionID string) error
	RemoveRolePermission(ctx context.Context, roleID, permissionID string) error

	GetUserRoles(ctx context.Context, userID string) ([]model.Role, error)
	GetRolesForUsers(ctx context.Context, userIDs []string) (map[string][]model.Role, error)
	GetUserPermissions(ctx context.Context, userID string) ([]model.Permission, error)
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID string) error
	RBACManager() *rbac.Manager
	RebuildRBAC(ctx context.Context) error
}

func (s *SQLiteStorage) CreatePermission(ctx context.Context, perm *model.Permission) error {
	if perm.ID == "" {
		perm.ID = newUUID()
	}
	if perm.CreatedAt.IsZero() {
		perm.CreatedAt = nowUTC()
	}

	query := `INSERT INTO permissions (id, name, resource, action, created_at) VALUES (?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query, perm.ID, perm.Name, perm.Resource, perm.Action, perm.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) GetPermission(ctx context.Context, id string) (*model.Permission, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	query := `SELECT id, name, resource, action, created_at FROM permissions WHERE id = ?`

	var perm model.Permission
	err := s.db.QueryRowContext(ctx, query, id).Scan(&perm.ID, &perm.Name, &perm.Resource, &perm.Action, &perm.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrPermissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &perm, nil
}

func (s *SQLiteStorage) GetPermissionByName(ctx context.Context, name string) (*model.Permission, error) {
	if name == "" {
		return nil, fmt.Errorf("permission name cannot be empty")
	}

	query := `SELECT id, name, resource, action, created_at FROM permissions WHERE name = ?`

	var perm model.Permission
	err := s.db.QueryRowContext(ctx, query, name).Scan(&perm.ID, &perm.Name, &perm.Resource, &perm.Action, &perm.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrPermissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &perm, nil
}

func (s *SQLiteStorage) ListPermissions(ctx context.Context, filter *model.PermissionFilter) ([]model.Permission, error) {
	query := `SELECT id, name, resource, action, created_at FROM permissions WHERE 1=1`
	var args []interface{}

	if filter != nil && filter.Resource != "" {
		query += " AND resource = ?"
		args = append(args, filter.Resource)
	}

	if filter != nil && filter.Action != "" {
		query += " AND action = ?"
		args = append(args, filter.Action)
	}

	query += " ORDER BY resource, action"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var perms []model.Permission
	for rows.Next() {
		var perm model.Permission
		if err := rows.Scan(&perm.ID, &perm.Name, &perm.Resource, &perm.Action, &perm.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		perms = append(perms, perm)
	}

	if perms == nil {
		perms = []model.Permission{}
	}

	return perms, nil
}

func (s *SQLiteStorage) DeletePermission(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	_, err := s.db.ExecContext(ctx, `DELETE FROM permissions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	s.refreshRBAC(ctx)
	return nil
}

func (s *SQLiteStorage) CreateRole(ctx context.Context, role *model.Role) error {
	if role.ID == "" {
		role.ID = newUUID()
	}
	if role.CreatedAt.IsZero() {
		role.CreatedAt = nowUTC()
	}
	if role.UpdatedAt.IsZero() {
		role.UpdatedAt = nowUTC()
	}

	query := `INSERT INTO roles (id, name, description, is_system, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query, role.ID, role.Name, role.Description, role.IsSystem, role.CreatedAt, role.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	s.refreshRBAC(ctx)
	return nil
}

func (s *SQLiteStorage) GetRole(ctx context.Context, id string) (*model.Role, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	query := `SELECT id, name, description, is_system, created_at, updated_at FROM roles WHERE id = ?`

	var role model.Role
	err := s.db.QueryRowContext(ctx, query, id).Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

func (s *SQLiteStorage) GetRoleByName(ctx context.Context, name string) (*model.Role, error) {
	if name == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	query := `SELECT id, name, description, is_system, created_at, updated_at FROM roles WHERE name = ?`

	var role model.Role
	err := s.db.QueryRowContext(ctx, query, name).Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

func (s *SQLiteStorage) ListRoles(ctx context.Context, filter *model.RoleFilter) ([]model.Role, error) {
	query := `SELECT id, name, description, is_system, created_at, updated_at FROM roles WHERE 1=1`
	var args []interface{}

	if filter != nil && filter.Name != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+filter.Name+"%")
	}

	if filter != nil && filter.IsSystem != nil {
		query += " AND is_system = ?"
		args = append(args, *filter.IsSystem)
	}

	query += " ORDER BY is_system DESC, name"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if roles == nil {
		roles = []model.Role{}
	}

	return roles, nil
}

func (s *SQLiteStorage) UpdateRole(ctx context.Context, role *model.Role) error {
	if role.ID == "" {
		return ErrInvalidID
	}

	role.UpdatedAt = nowUTC()

	query := `UPDATE roles SET description = ?, updated_at = ? WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, role.Description, role.UpdatedAt, role.ID)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrRoleNotFound
	}

	return nil
}

func (s *SQLiteStorage) DeleteRole(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	var role model.Role
	err := s.db.QueryRowContext(ctx, `SELECT is_system FROM roles WHERE id = ?`, id).Scan(&role.IsSystem)
	if err == sql.ErrNoRows {
		return ErrRoleNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check role: %w", err)
	}

	if role.IsSystem {
		return fmt.Errorf("cannot delete system role")
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM roles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	s.refreshRBAC(ctx)
	return nil
}

func (s *SQLiteStorage) GetRolePermissions(ctx context.Context, roleID string) ([]model.Permission, error) {
	if roleID == "" {
		return nil, ErrInvalidID
	}

	query := `
		SELECT p.id, p.name, p.resource, p.action, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = ?
		ORDER BY p.resource, p.action
	`

	rows, err := s.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var perms []model.Permission
	for rows.Next() {
		var perm model.Permission
		if err := rows.Scan(&perm.ID, &perm.Name, &perm.Resource, &perm.Action, &perm.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		perms = append(perms, perm)
	}

	if perms == nil {
		perms = []model.Permission{}
	}

	return perms, nil
}

func (s *SQLiteStorage) SetRolePermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	if roleID == "" {
		return ErrInvalidID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = ?`, roleID)
	if err != nil {
		return fmt.Errorf("failed to clear role permissions: %w", err)
	}

	now := nowUTC()
	for _, permID := range permissionIDs {
		_, err = tx.ExecContext(ctx, `INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)`,
			roleID, permID, now)
		if err != nil {
			return fmt.Errorf("failed to add role permission: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	s.refreshRBAC(ctx)
	return nil
}

func (s *SQLiteStorage) AddRolePermission(ctx context.Context, roleID, permissionID string) error {
	if roleID == "" || permissionID == "" {
		return ErrInvalidID
	}

	query := `INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query, roleID, permissionID, nowUTC())
	if err != nil {
		return fmt.Errorf("failed to add role permission: %w", err)
	}

	s.refreshRBAC(ctx)
	return nil
}

func (s *SQLiteStorage) RemoveRolePermission(ctx context.Context, roleID, permissionID string) error {
	if roleID == "" || permissionID == "" {
		return ErrInvalidID
	}

	_, err := s.db.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = ? AND permission_id = ?`, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove role permission: %w", err)
	}

	s.refreshRBAC(ctx)
	return nil
}

func (s *SQLiteStorage) GetUserRoles(ctx context.Context, userID string) ([]model.Role, error) {
	if userID == "" {
		return nil, ErrInvalidID
	}

	query := `
		SELECT r.id, r.name, r.description, r.is_system, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = ?
		ORDER BY r.name
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if roles == nil {
		roles = []model.Role{}
	}

	return roles, nil
}

// GetRolesForUsers returns each user's roles, by name, in one query, so a
// list of users doesn't need a query per user. Users without roles have an
// empty list.
func (s *SQLiteStorage) GetRolesForUsers(ctx context.Context, userIDs []string) (map[string][]model.Role, error) {
	result := make(map[string][]model.Role, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	args := make([]any, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
		result[id] = []model.Role{}
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT ur.user_id, r.id, r.name, r.description, r.is_system, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id IN (?`+strings.Repeat(",?", len(userIDs)-1)+`)
		ORDER BY r.name
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles for users: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var userID string
		var role model.Role
		if err := rows.Scan(&userID, &role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		result[userID] = append(result[userID], role)
	}
	return result, rows.Err()
}

// GetUserPermissions returns the permissions the user's roles grant, as
// resolved by the in-memory RBAC manager, so it always agrees with what
// permission checks enforce.
func (s *SQLiteStorage) GetUserPermissions(ctx context.Context, userID string) ([]model.Permission, error) {
	if userID == "" {
		return nil, ErrInvalidID
	}

	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	granted := make(map[string]struct{})
	for _, p := range s.rbacManager.AllPermissions(RBACRoles(roles)) {
		granted[string(p)] = struct{}{}
	}

	perms := []model.Permission{}
	if len(granted) == 0 {
		return perms, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id, name, resource, action, created_at FROM permissions ORDER BY resource, action`)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var perm model.Permission
		if err := rows.Scan(&perm.ID, &perm.Name, &perm.Resource, &perm.Action, &perm.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		if _, ok := granted[perm.Resource+":"+perm.Action]; ok {
			perms = append(perms, perm)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	return perms, nil
}

func (s *SQLiteStorage) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	if userID == "" || roleID == "" {
		return ErrInvalidID
	}

	return s.changeUserRoles(ctx, userID, `INSERT OR IGNORE INTO user_roles (user_id, role_id, created_at) VALUES (?, ?, ?)`,
		userID, roleID, nowUTC())
}

func (s *SQLiteStorage) RemoveRoleFromUser(ctx context.Context, userID, roleID string) error {
	if userID == "" || roleID == "" {
		return ErrInvalidID
	}

	return s.changeUserRoles(ctx, userID, `DELETE FROM user_roles WHERE user_id = ? AND role_id = ?`, userID, roleID)
}

// changeUserRoles runs a change to a user's roles and updates the user's
// is_admin to match, in one transaction.
func (s *SQLiteStorage) changeUserRoles(ctx context.Context, userID, query string, args ...any) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to change user roles: %w", err)
	}
	if _, err := tx.ExecContext(ctx, syncIsAdminSQL+` WHERE id = ?`, userID); err != nil {
		return fmt.Errorf("failed to update is_admin: %w", err)
	}
	return tx.Commit()
}

func (s *SQLiteStorage) RBACManager() *rbac.Manager { return s.rbacManager }

// RBACRoles converts a user's assigned roles into the form the RBAC manager
// checks against.
func RBACRoles(roles []model.Role) rbac.Roles {
	names := make(rbac.Roles, len(roles))
	for i, r := range roles {
		names[i] = rbac.Role(r.Name)
	}
	return names
}

// initRBAC creates the in-memory RBAC manager from the database. It is called
// once by the storage constructors. Wildcards are disabled so stored
// permissions always match exactly, as the original SQL check did.
func (s *SQLiteStorage) initRBAC(ctx context.Context) error {
	m, err := rbac.NewFromLoader(ctx, rbac.RoleLoaderFunc(s.loadRoleDefinitions), rbac.WithoutWildcards())
	if err != nil {
		return fmt.Errorf("failed to initialize RBAC manager: %w", err)
	}
	s.rbacManager = m
	return nil
}

// RebuildRBAC reloads role→permission definitions from the database into the
// in-memory RBAC manager. go-rbac serializes rebuilds (including the load), so
// a slower, older load can never overwrite a newer one, and a failed rebuild
// leaves the previous definitions in place.
func (s *SQLiteStorage) RebuildRBAC(ctx context.Context) error {
	return s.rbacManager.RebuildFromLoader(ctx, rbac.RoleLoaderFunc(s.loadRoleDefinitions))
}

// refreshRBAC rebuilds the RBAC manager after a role/permission mutation has
// been committed. The rebuild ignores cancellation of the request context: the
// write has already happened, so the in-memory state must follow it even if
// the client has gone away. A failure is logged rather than returned, because
// reporting an error for a change that was persisted would mislead callers.
func (s *SQLiteStorage) refreshRBAC(ctx context.Context) {
	if err := s.RebuildRBAC(context.WithoutCancel(ctx)); err != nil {
		log.Error("RBAC: failed to rebuild permissions after change; permission checks may be stale until the next successful rebuild", "error", err)
	}
}

// loadRoleDefinitions implements rbac.RoleLoader. Roles without permissions
// are included so they are still known to the manager.
func (s *SQLiteStorage) loadRoleDefinitions(ctx context.Context) (map[rbac.Role]rbac.RoleDefinition, error) {
	query := `
		SELECT r.name, p.resource, p.action
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN permissions p ON p.id = rp.permission_id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load role definitions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	defs := make(map[rbac.Role]rbac.RoleDefinition)
	for rows.Next() {
		var roleName string
		var resource, action sql.NullString
		if err := rows.Scan(&roleName, &resource, &action); err != nil {
			return nil, fmt.Errorf("failed to scan role definition: %w", err)
		}
		def := defs[rbac.Role(roleName)]
		if resource.Valid && action.Valid {
			def.Permissions = append(def.Permissions, rbac.Permission(resource.String+":"+action.String))
		}
		defs[rbac.Role(roleName)] = def
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to load role definitions: %w", err)
	}

	return defs, nil
}
