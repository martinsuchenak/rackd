package credentials

import (
	"context"
	"testing"

	"github.com/martinsuchenak/xdal/pkg/xdal"

	corestorage "github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/storage/dialecttest"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// The store works on PostgreSQL and MySQL: create, read, list, update, and a
// delete that clears the credential from the scan profiles that use it.
// SQLite is covered by the other tests here.
func TestDialectCredentialStore(t *testing.T) {
	dialecttest.Run(t, corestorage.SchemaMigrationNamespace, corestorage.SchemaMigrations(), func(t *testing.T, db *dialecttest.DB) {
		if db.Name == xdal.DialectSQLite {
			t.Skip("SQLite: covered by the store's other tests")
		}
		ctx := context.Background()
		store, err := NewStorage(db.DB, make([]byte, 32))
		if err != nil {
			t.Fatal(err)
		}

		cred := &model.Credential{Name: "snmp", Type: "snmp_v3", SNMPV3User: "u", SNMPV3Auth: "a", SNMPV3Priv: "p"}
		if err := store.Create(cred); err != nil {
			t.Fatalf("create: %v", err)
		}
		got, err := store.Get(cred.ID)
		if err != nil || got.SNMPV3User != "u" || got.SNMPV3Priv != "p" {
			t.Fatalf("get = %+v, %v", got, err)
		}
		cred.Description = "updated"
		if err := store.Update(cred); err != nil {
			t.Fatalf("update: %v", err)
		}
		if list, err := store.List(""); err != nil || len(list) != 1 || list[0].Description != "updated" {
			t.Fatalf("list = %+v, %v", list, err)
		}

		q, args, err := db.QueryBuilder().Insert("scan_profiles").Set("id", "p1").Set("name", "with-cred").
			Set("scan_type", "full").Set("snmp_credential_id", cred.ID).Build()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.DB.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
		if err := store.Delete(cred.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if _, err := store.Get(cred.ID); err != ErrCredentialNotFound {
			t.Errorf("get after delete: %v", err)
		}
		q, args, _ = db.QueryBuilder().Select("snmp_credential_id").From("scan_profiles").Where("id = ?", "p1").Build()
		var ref *string
		if err := db.DB.QueryRow(ctx, q, args...).Scan(&ref); err != nil || ref != nil {
			t.Errorf("profile still refers to the deleted credential: %v, %v", ref, err)
		}
		if err := store.Delete(cred.ID); err != ErrCredentialNotFound {
			t.Errorf("second delete: %v", err)
		}
	})
}
