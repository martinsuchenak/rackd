package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
)

func TestCreateUser(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	passwordHash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := &model.User{
		Username:     "testuser",
		Email:        "test@example.com",
		FullName:     "Test User",
		PasswordHash: passwordHash,
		IsActive:     true,
		IsAdmin:      false,
	}

	err = db.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if user.ID == "" {
		t.Error("CreateUser() did not set ID")
	}

	if user.CreatedAt.IsZero() {
		t.Error("CreateUser() did not set CreatedAt")
	}

	if user.UpdatedAt.IsZero() {
		t.Error("CreateUser() did not set UpdatedAt")
	}
}

func TestGetUser(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	created, err := createUser(t, db, "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	user, err := db.GetUser(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}

	if user.ID != created.ID {
		t.Errorf("GetUser() ID = %v, want %v", user.ID, created.ID)
	}

	if user.Username != "testuser" {
		t.Errorf("GetUser() Username = %v, want testuser", user.Username)
	}

	if user.Email != "test@example.com" {
		t.Errorf("GetUser() Email = %v, want test@example.com", user.Email)
	}
}

func TestGetUserNotFound(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	_, err := db.GetUser(context.Background(), "nonexistent")
	if err == nil {
		t.Error("GetUser() expected error, got nil")
	}
}

func TestGetUserByID(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	created, _ := createUser(t, db, "testuser", "test@example.com")

	user, err := db.GetUser(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}

	if user.ID != created.ID {
		t.Errorf("GetUser() ID = %v, want %v", user.ID, created.ID)
	}
}

func TestGetUserByUsername(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	created, _ := createUser(t, db, "testuser", "test@example.com")

	user, err := db.GetUserByUsername(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}

	if user.ID != created.ID {
		t.Errorf("GetUserByUsername() ID = %v, want %v", user.ID, created.ID)
	}
}

func TestGetUserByEmail(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	created, _ := createUser(t, db, "testuser", "test@example.com")

	user, err := db.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v", err)
	}

	if user.ID != created.ID {
		t.Errorf("GetUserByEmail() ID = %v, want %v", user.ID, created.ID)
	}
}

func TestListUsers(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	_, _ = createUser(t, db, "user1", "user1@example.com")
	_, _ = createUser(t, db, "user2", "user2@example.com")
	_, _ = createUser(t, db, "user3", "user3@example.com")

	users, err := db.ListUsers(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}

	if len(users) < 3 {
		t.Errorf("ListUsers() returned %d users, want at least 3", len(users))
	}
}

func TestListUsersFilter(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	_, _ = createUser(t, db, "admin1", "admin@example.com")
	_, _ = createUser(t, db, "admin2", "admin2@example.com")
	_, _ = createUser(t, db, "user1", "user@example.com")

	filter := &model.UserFilter{
		Username: "admin",
	}

	users, err := db.ListUsers(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}

	if len(users) != 2 {
		t.Errorf("ListUsers() with filter returned %d users, want 2", len(users))
	}

	for _, user := range users {
		if user.Username != "admin1" && user.Username != "admin2" {
			t.Errorf("ListUsers() with filter returned unexpected user: %v", user.Username)
		}
	}
}

func TestUpdateUser(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	created, _ := createUser(t, db, "testuser", "test@example.com")
	newHash, err := auth.HashPassword("updatedpassword123")
	if err != nil {
		t.Fatalf("Failed to hash updated password: %v", err)
	}

	created.Username = "updateduser"
	created.Email = "updated@example.com"
	created.FullName = "Updated Name"
	created.PasswordHash = newHash

	err = db.UpdateUser(context.Background(), created)
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	user, err := db.GetUser(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetUser() after update error = %v", err)
	}

	if user.Email != "updated@example.com" {
		t.Errorf("UpdateUser() Email = %v, want updated@example.com", user.Email)
	}

	if user.Username != "updateduser" {
		t.Errorf("UpdateUser() Username = %v, want updateduser", user.Username)
	}

	if user.FullName != "Updated Name" {
		t.Errorf("UpdateUser() FullName = %v, want Updated Name", user.FullName)
	}

	if user.PasswordHash != newHash {
		t.Error("UpdateUser() did not update password hash")
	}

	if err := auth.VerifyPassword(user.PasswordHash, "updatedpassword123"); err != nil {
		t.Errorf("UpdateUser() password verification failed: %v", err)
	}
}

func TestDeleteUser(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	created, _ := createUser(t, db, "testuser", "test@example.com")

	err := db.DeleteUser(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	_, err = db.GetUser(context.Background(), created.ID)
	if err == nil {
		t.Error("GetUser() after DeleteUser() expected error, got nil")
	}
}

func TestUpdateUserLastLogin(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	created, _ := createUser(t, db, "testuser", "test@example.com")

	err := db.UpdateUserLastLogin(context.Background(), created.ID, created.CreatedAt)
	if err != nil {
		t.Fatalf("UpdateUserLastLogin() error = %v", err)
	}

	user, err := db.GetUser(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetUser() after update error = %v", err)
	}

	if user.LastLoginAt == nil {
		t.Error("UpdateUserLastLogin() did not set LastLoginAt")
	}
}

func TestUpdateUserPassword(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	created, _ := createUser(t, db, "testuser", "test@example.com")

	newHash, err := auth.HashPassword("newpassword123")
	if err != nil {
		t.Fatalf("Failed to hash new password: %v", err)
	}

	err = db.UpdateUserPassword(context.Background(), created.ID, newHash)
	if err != nil {
		t.Fatalf("UpdateUserPassword() error = %v", err)
	}

	user, err := db.GetUser(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetUser() after update error = %v", err)
	}

	if user.PasswordHash != newHash {
		t.Error("UpdateUserPassword() did not update password hash")
	}

	if err := auth.VerifyPassword(user.PasswordHash, "newpassword123"); err != nil {
		t.Errorf("UpdateUserPassword() new password verification failed: %v", err)
	}
}

func createUser(t *testing.T, db *SQLiteStorage, username, email string) (*model.User, error) {
	passwordHash, err := auth.HashPassword("password123")
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		IsActive:     true,
		IsAdmin:      false,
	}

	err = db.CreateUser(context.Background(), user)
	return user, err
}
