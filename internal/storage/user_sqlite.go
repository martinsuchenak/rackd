package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
)

type UserStorage interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUser(ctx context.Context, id string) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	ListUsers(ctx context.Context, filter *model.UserFilter) ([]model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id string) error
	UpdateUserLastLogin(ctx context.Context, id string, lastLogin time.Time) error
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error
	UserCount(ctx context.Context) (int, error)
	CreateInitialAdmin(ctx context.Context, username, email, fullName, password string) error
}

func (s *SQLiteStorage) CreateUser(ctx context.Context, user *model.User) error {
	if user.ID == "" {
		user.ID = newUUID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = nowUTC()
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = nowUTC()
	}

	query := `INSERT INTO users (id, username, email, full_name, password_hash, is_active, is_admin, created_at, updated_at, last_login_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Email, user.FullName,
		user.PasswordHash, user.IsActive, user.IsAdmin,
		user.CreatedAt, user.UpdatedAt, user.LastLoginAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) GetUser(ctx context.Context, id string) (*model.User, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	query := `SELECT id, username, email, full_name, password_hash, is_active, is_admin, created_at, updated_at, last_login_at 
	          FROM users WHERE id = ?`

	var user model.User
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.FullName,
		&user.PasswordHash, &user.IsActive, &user.IsAdmin,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}

func (s *SQLiteStorage) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	query := `SELECT id, username, email, full_name, password_hash, is_active, is_admin, created_at, updated_at, last_login_at 
	          FROM users WHERE username = ?`

	var user model.User
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.FullName,
		&user.PasswordHash, &user.IsActive, &user.IsAdmin,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}

func (s *SQLiteStorage) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	query := `SELECT id, username, email, full_name, password_hash, is_active, is_admin, created_at, updated_at, last_login_at 
	          FROM users WHERE email = ?`

	var user model.User
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.FullName,
		&user.PasswordHash, &user.IsActive, &user.IsAdmin,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}

func (s *SQLiteStorage) ListUsers(ctx context.Context, filter *model.UserFilter) ([]model.User, error) {
	query := `SELECT id, username, email, full_name, password_hash, is_active, is_admin, created_at, updated_at, last_login_at 
	          FROM users WHERE 1=1`
	var args []interface{}

	if filter != nil && filter.Username != "" {
		query += " AND username LIKE ?"
		args = append(args, "%"+filter.Username+"%")
	}

	if filter != nil && filter.Email != "" {
		query += " AND email LIKE ?"
		args = append(args, "%"+filter.Email+"%")
	}

	if filter != nil && filter.IsActive != nil {
		query += " AND is_active = ?"
		args = append(args, *filter.IsActive)
	}

	if filter != nil && filter.IsAdmin != nil {
		query += " AND is_admin = ?"
		args = append(args, *filter.IsAdmin)
	}

	query += " ORDER BY created_at DESC"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		var lastLoginAt sql.NullTime

		if err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.FullName,
			&user.PasswordHash, &user.IsActive, &user.IsAdmin,
			&user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}

	if users == nil {
		users = []model.User{}
	}

	return users, nil
}

func (s *SQLiteStorage) UpdateUser(ctx context.Context, user *model.User) error {
	if user.ID == "" {
		return ErrInvalidID
	}

	user.UpdatedAt = nowUTC()

	query := `UPDATE users SET username = ?, email = ?, full_name = ?, password_hash = ?, is_active = ?, is_admin = ?, updated_at = ? 
	          WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query,
		user.Username, user.Email, user.FullName, user.PasswordHash, user.IsActive, user.IsAdmin,
		user.UpdatedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (s *SQLiteStorage) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if !exists {
		return ErrUserNotFound
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) UpdateUserLastLogin(ctx context.Context, id string, lastLogin time.Time) error {
	if id == "" {
		return ErrInvalidID
	}

	query := `UPDATE users SET last_login_at = ? WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query, lastLogin, id)
	if err != nil {
		return fmt.Errorf("failed to update user last login: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	if id == "" {
		return ErrInvalidID
	}

	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query, passwordHash, nowUTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) UserCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

func (s *SQLiteStorage) CreateInitialAdmin(ctx context.Context, username, email, fullName, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required for initial admin")
	}

	if len(password) < 8 {
		return fmt.Errorf("initial admin password must be at least 8 characters")
	}

	existingUser, err := s.GetUserByUsername(ctx, username)
	if err == nil && existingUser != nil {
		return fmt.Errorf("user '%s' already exists", username)
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash initial admin password: %w", err)
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		FullName:     fullName,
		PasswordHash: passwordHash,
		IsActive:     true,
		IsAdmin:      true,
	}

	if err := s.CreateUser(ctx, user); err != nil {
		return err
	}

	// Assign the admin role so RBAC permissions are enforced
	adminRole, err := s.GetRoleByName(ctx, "admin")
	if err != nil {
		return fmt.Errorf("failed to find admin role: %w", err)
	}

	if err := s.AssignRoleToUser(ctx, user.ID, adminRole.ID); err != nil {
		return fmt.Errorf("failed to assign admin role: %w", err)
	}

	return nil
}
