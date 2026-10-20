package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Listing users loads their roles in batches, so a long unpaginated list
// gets every user's roles and stays within SQLite's parameter limit.
func TestGetRolesForUsersAcrossBatches(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	viewer, err := s.GetRoleByName(ctx, "viewer")
	if err != nil {
		t.Fatal(err)
	}

	var ids []string
	for i := range 1001 {
		u := &model.User{Username: fmt.Sprintf("u%d", i), Email: fmt.Sprintf("u%d@example.test", i), PasswordHash: "x", IsActive: true}
		if err := s.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
		if i%2 == 0 {
			if err := s.AssignRoleToUser(ctx, u.ID, viewer.ID); err != nil {
				t.Fatal(err)
			}
		}
		ids = append(ids, u.ID)
	}

	roles, err := s.GetRolesForUsers(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range ids {
		got := roles[id]
		if i%2 == 0 && (len(got) != 1 || got[0].Name != "viewer") {
			t.Fatalf("user %d roles = %v, want [viewer]", i, got)
		}
		if i%2 == 1 && (got == nil || len(got) != 0) {
			t.Fatalf("user %d roles = %v, want an empty list", i, got)
		}
	}
}
