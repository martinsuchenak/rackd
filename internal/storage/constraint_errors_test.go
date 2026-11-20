package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Constraint violations are recognised, with their column, on every backend:
// the API turns them into "already exists", "is required" and "refers to a
// missing record" errors.
func TestConstraintErrorsOnEveryBackend(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	def := &model.CustomFieldDefinition{Name: "Rack", Key: "rack", Type: model.CustomFieldTypeText}
	if err := s.CreateCustomFieldDefinition(ctx, def); err != nil {
		t.Fatal(err)
	}
	dc := &model.Datacenter{Name: "dc"}
	if err := s.CreateDatacenter(ctx, dc); err != nil {
		t.Fatal(err)
	}

	// A unique column, written directly so the storage method's own check
	// doesn't get there first.
	_, err := s.q.ExecContext(ctx, `INSERT INTO custom_field_definitions (id, name, `+s.q.ident("key")+`, type, required, options, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, newUUID(), "Rack 2", "rack", "text", false, "[]", "", nowUTC(), nowUTC())
	if field, ok := DuplicateField(err); !ok || field != "key" {
		t.Errorf("duplicate key: DuplicateField(%v) = %q, %v", err, field, ok)
	}

	_, err = s.q.ExecContext(ctx, `INSERT INTO networks (id, name, subnet) VALUES (?, ?, ?)`, newUUID(), nil, "10.0.0.0/24")
	if field, ok := MissingField(err); !ok || field != "name" {
		t.Errorf("missing name: MissingField(%v) = %q, %v", err, field, ok)
	}

	_, err = s.q.ExecContext(ctx, `INSERT INTO networks (id, name, subnet, datacenter_id) VALUES (?, ?, ?, ?)`,
		newUUID(), "n", "10.0.0.0/24", "no-such-datacenter")
	if !IsForeignKeyFailure(err) {
		t.Errorf("missing datacenter: IsForeignKeyFailure(%v) = false", err)
	}

	for _, err := range []error{nil, context.Canceled} {
		if _, ok := DuplicateField(err); ok {
			t.Errorf("DuplicateField(%v) = true", err)
		}
		if IsForeignKeyFailure(err) {
			t.Errorf("IsForeignKeyFailure(%v) = true", err)
		}
	}
}
