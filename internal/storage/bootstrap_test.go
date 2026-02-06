package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
)

func init() {
	log.Init("text", "info", nil)
}

func TestBootstrapInitialAdmin(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	cfg := &config.Config{
		InitialAdminUsername: "testadmin",
		InitialAdminPassword: "testpassword123",
		InitialAdminEmail:    "admin@test.com",
		InitialAdminFullName: "Test Admin",
		SessionTTL:           3600,
	}

	err := BootstrapInitialAdmin(db, cfg, nil)
	if err != nil {
		t.Fatalf("BootstrapInitialAdmin() error = %v", err)
	}

	users, err := db.ListUsers(nil)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}

	if len(users) != 1 {
		t.Errorf("ListUsers() returned %d users, want 1", len(users))
	}

	if users[0].Username != "testadmin" {
		t.Errorf("ListUsers()[0].Username = %v, want testadmin", users[0].Username)
	}

	if !users[0].IsAdmin {
		t.Error("ListUsers()[0].IsAdmin = false, want true")
	}
}

func TestBootstrapInitialAdminSkipsIfExists(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	_, _ = createTestUser(t, db, "existinguser", "existing@test.com")

	cfg := &config.Config{
		InitialAdminUsername: "testadmin",
		InitialAdminPassword: "testpassword123",
		SessionTTL:           3600,
	}

	err := BootstrapInitialAdmin(db, cfg, nil)
	if err != nil {
		t.Fatalf("BootstrapInitialAdmin() error = %v", err)
	}

	users, err := db.ListUsers(nil)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}

	if len(users) != 1 {
		t.Errorf("ListUsers() returned %d users, want 1 (should not create admin if users exist)", len(users))
	}

	if users[0].Username != "existinguser" {
		t.Errorf("ListUsers()[0].Username = %v, want existinguser", users[0].Username)
	}
}

func TestBootstrapInitialAdminNoConfig(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	cfg := &config.Config{
		InitialAdminUsername: "",
		InitialAdminPassword: "",
		SessionTTL:           3600,
	}

	err := BootstrapInitialAdmin(db, cfg, nil)
	if err != nil {
		t.Fatalf("BootstrapInitialAdmin() should not error when no config, got: %v", err)
	}

	users, err := db.ListUsers(nil)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}

	if len(users) != 0 {
		t.Errorf("ListUsers() returned %d users, want 0 (should not create admin without config)", len(users))
	}
}

func TestCreateInitialAdmin(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	err := db.CreateInitialAdmin("testadmin", "admin@test.com", "Test Admin", "testpassword123")
	if err != nil {
		t.Fatalf("CreateInitialAdmin() error = %v", err)
	}
	if err != nil {
		t.Fatalf("CreateInitialAdmin() error = %v", err)
	}

	user, err := db.GetUserByUsername("testadmin")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}

	if user.Email != "admin@test.com" {
		t.Errorf("GetUserByUsername().Email = %v, want admin@test.com", user.Email)
	}

	if user.FullName != "Test Admin" {
		t.Errorf("GetUserByUsername().FullName = %v, want Test Admin", user.FullName)
	}

	if !user.IsAdmin {
		t.Error("GetUserByUsername().IsAdmin = false, want true")
	}

	if !user.IsActive {
		t.Error("GetUserByUsername().IsActive = false, want true")
	}
}

func TestCreateInitialAdminAlreadyExists(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	err := db.CreateInitialAdmin("testadmin", "admin@test.com", "Test Admin", "testpassword123")
	if err != nil {
		t.Fatalf("First CreateInitialAdmin() error = %v", err)
	}

	err = db.CreateInitialAdmin("testadmin", "admin2@test.com", "Test Admin 2", "testpassword123")
	if err == nil {
		t.Error("CreateInitialAdmin() should error when user already exists")
	}
}

func TestCreateInitialAdminInvalidPassword(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	err := db.CreateInitialAdmin("testadmin", "admin@test.com", "Test Admin", "short")
	if err == nil {
		t.Error("CreateInitialAdmin() should error for short password")
	}
}

func TestCreateInitialAdminMissingFields(t *testing.T) {
	db := newTestStorage(t)
	defer db.Close()

	err := db.CreateInitialAdmin("", "admin@test.com", "Test Admin", "testpassword123")
	if err == nil {
		t.Error("CreateInitialAdmin() should error for missing username")
	}

	err = db.CreateInitialAdmin("testadmin", "admin@test.com", "Test Admin", "")
	if err == nil {
		t.Error("CreateInitialAdmin() should error for missing password")
	}
}

func createTestUser(t *testing.T, db *SQLiteStorage, username, email string) (*model.User, error) {
	t.Helper()
	ctx := context.Background()

	user := &model.User{
		Username: username,
		Email:    email,
		IsActive: true,
		IsAdmin:  false,
	}

	err := db.CreateUser(ctx, user)
	return user, err
}
