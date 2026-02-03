package credentials

import (
	"crypto/rand"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
	"github.com/martinsuchenak/rackd/internal/model"
)

func setupTestDB(t *testing.T) (*SQLiteStorage, func()) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	storage, err := NewSQLiteStorage(db, key)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		db.Close()
	}

	return storage, cleanup
}

func TestSQLiteStorage_Create(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	cred := &model.Credential{
		Name:          "Test SNMP",
		Type:          "snmp_v2c",
		SNMPCommunity: "public",
		Description:   "Test credential",
	}

	err := storage.Create(cred)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if cred.ID == "" {
		t.Error("expected ID to be generated")
	}
	if cred.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if cred.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestSQLiteStorage_CreateInvalid(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	cred := &model.Credential{
		Name: "Invalid",
		Type: "snmp_v2c",
	}

	err := storage.Create(cred)
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestSQLiteStorage_Get(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	cred := &model.Credential{
		Name:          "Test SNMP",
		Type:          "snmp_v2c",
		SNMPCommunity: "secret",
	}

	if err := storage.Create(cred); err != nil {
		t.Fatal(err)
	}

	retrieved, err := storage.Get(cred.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.ID != cred.ID {
		t.Errorf("ID = %v, want %v", retrieved.ID, cred.ID)
	}
	if retrieved.Name != cred.Name {
		t.Errorf("Name = %v, want %v", retrieved.Name, cred.Name)
	}
	if retrieved.SNMPCommunity != cred.SNMPCommunity {
		t.Errorf("SNMPCommunity = %v, want %v", retrieved.SNMPCommunity, cred.SNMPCommunity)
	}
}

func TestSQLiteStorage_GetNotFound(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := storage.Get("nonexistent")
	if err != ErrCredentialNotFound {
		t.Errorf("expected ErrCredentialNotFound, got %v", err)
	}
}

func TestSQLiteStorage_Update(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	cred := &model.Credential{
		Name:          "Original",
		Type:          "snmp_v2c",
		SNMPCommunity: "public",
	}

	if err := storage.Create(cred); err != nil {
		t.Fatal(err)
	}

	cred.Name = "Updated"
	cred.SNMPCommunity = "private"

	if err := storage.Update(cred); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	retrieved, err := storage.Get(cred.ID)
	if err != nil {
		t.Fatal(err)
	}

	if retrieved.Name != "Updated" {
		t.Errorf("Name = %v, want Updated", retrieved.Name)
	}
	if retrieved.SNMPCommunity != "private" {
		t.Errorf("SNMPCommunity = %v, want private", retrieved.SNMPCommunity)
	}
}

func TestSQLiteStorage_UpdateNotFound(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	cred := &model.Credential{
		ID:            "nonexistent",
		Name:          "Test",
		Type:          "snmp_v2c",
		SNMPCommunity: "public",
	}

	err := storage.Update(cred)
	if err != ErrCredentialNotFound {
		t.Errorf("expected ErrCredentialNotFound, got %v", err)
	}
}

func TestSQLiteStorage_List(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	creds := []*model.Credential{
		{Name: "Cred1", Type: "snmp_v2c", SNMPCommunity: "public", DatacenterID: "dc1"},
		{Name: "Cred2", Type: "snmp_v2c", SNMPCommunity: "public", DatacenterID: "dc2"},
		{Name: "Cred3", Type: "snmp_v2c", SNMPCommunity: "public"},
	}

	for _, c := range creds {
		if err := storage.Create(c); err != nil {
			t.Fatal(err)
		}
	}

	all, err := storage.List("")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List() returned %d credentials, want 3", len(all))
	}

	dc1, err := storage.List("dc1")
	if err != nil {
		t.Fatal(err)
	}
	if len(dc1) != 2 {
		t.Errorf("List(dc1) returned %d credentials, want 2", len(dc1))
	}
}

func TestSQLiteStorage_Delete(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	cred := &model.Credential{
		Name:          "Test",
		Type:          "snmp_v2c",
		SNMPCommunity: "public",
	}

	if err := storage.Create(cred); err != nil {
		t.Fatal(err)
	}

	if err := storage.Delete(cred.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := storage.Get(cred.ID)
	if err != ErrCredentialNotFound {
		t.Errorf("expected ErrCredentialNotFound after delete, got %v", err)
	}
}

func TestSQLiteStorage_DeleteNotFound(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	err := storage.Delete("nonexistent")
	if err != ErrCredentialNotFound {
		t.Errorf("expected ErrCredentialNotFound, got %v", err)
	}
}

func TestSQLiteStorage_EncryptionRoundtrip(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name string
		cred *model.Credential
	}{
		{
			name: "snmp_v2c",
			cred: &model.Credential{
				Name:          "SNMP v2c",
				Type:          "snmp_v2c",
				SNMPCommunity: "secret_community",
			},
		},
		{
			name: "snmp_v3",
			cred: &model.Credential{
				Name:       "SNMP v3",
				Type:       "snmp_v3",
				SNMPV3User: "admin",
				SNMPV3Auth: "auth_password",
				SNMPV3Priv: "priv_password",
			},
		},
		{
			name: "ssh_key",
			cred: &model.Credential{
				Name:        "SSH Key",
				Type:        "ssh_key",
				SSHUsername: "root",
				SSHKeyID:    "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := storage.Create(tt.cred); err != nil {
				t.Fatal(err)
			}

			retrieved, err := storage.Get(tt.cred.ID)
			if err != nil {
				t.Fatal(err)
			}

			if retrieved.SNMPCommunity != tt.cred.SNMPCommunity {
				t.Errorf("SNMPCommunity mismatch")
			}
			if retrieved.SNMPV3User != tt.cred.SNMPV3User {
				t.Errorf("SNMPV3User mismatch")
			}
			if retrieved.SNMPV3Auth != tt.cred.SNMPV3Auth {
				t.Errorf("SNMPV3Auth mismatch")
			}
			if retrieved.SNMPV3Priv != tt.cred.SNMPV3Priv {
				t.Errorf("SNMPV3Priv mismatch")
			}
			if retrieved.SSHKeyID != tt.cred.SSHKeyID {
				t.Errorf("SSHKeyID mismatch")
			}
		})
	}
}
