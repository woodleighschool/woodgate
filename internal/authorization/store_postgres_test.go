//go:build postgres

package authorization

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestEffectivePermissionsIncludesGroupRoles(t *testing.T) {
	database, ctx := testdb.Open(t)
	service := NewService(NewStore(database))

	var userID int64
	if err := database.QueryRow(ctx, `
INSERT INTO users (email, name, access_enabled, source)
VALUES ('group-member@example.invalid', 'Group Member', true, 'local')
RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("create user: %v", err)
	}

	var groupID int64
	if err := database.QueryRow(ctx, `
INSERT INTO directory_groups (source, external_id, display_name)
VALUES ('entra', 'group-external-id', 'Directory Group')
RETURNING id`).Scan(&groupID); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := database.Exec(ctx, `
INSERT INTO directory_group_memberships (user_id, group_id)
VALUES ($1, $2)`, userID, groupID); err != nil {
		t.Fatalf("create group membership: %v", err)
	}

	role, err := service.CreateRole(ctx, RoleMutation{
		Key:  "group-users-viewer",
		Name: "Group users viewer",
		Permissions: map[Resource]Access{
			"users": View,
		},
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := service.ReplaceAssignments(ctx, AssignmentMutation{
		SubjectKind: SubjectGroup,
		SubjectID:   groupID,
		RoleIDs:     []int64{role.ID},
	}); err != nil {
		t.Fatalf("assign group role: %v", err)
	}

	permissions, err := service.EffectivePermissions(ctx, userID)
	if err != nil {
		t.Fatalf("get effective permissions: %v", err)
	}
	if got := permissions["users"]; got != View {
		t.Fatalf("users permission = %q, want %q", got, View)
	}
	if got := permissions["groups"]; got != None {
		t.Fatalf("groups permission = %q, want %q", got, None)
	}
	if allowed, err := service.Can(ctx, userID, "users", Edit); err != nil {
		t.Fatalf("check edit access: %v", err)
	} else if allowed {
		t.Fatal("edit access allowed, want denied")
	}
}

func TestBuiltinRoleCannotBeChangedOrDeleted(t *testing.T) {
	database, ctx := testdb.Open(t)
	service := NewService(NewStore(database))

	roles, err := service.ListRoles(ctx)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	var owner *Role
	for i := range roles {
		if roles[i].Key == "owner" {
			owner = &roles[i]
			break
		}
	}
	if owner == nil {
		t.Fatal("owner role not found")
	}

	_, err = service.UpdateRole(ctx, owner.ID, RoleMutation{
		Name: "Changed owner",
		Permissions: map[Resource]Access{
			"users": View,
		},
	})
	if !errors.Is(err, fault.ErrConflict) {
		t.Fatalf("update builtin role error = %v, want %v", err, fault.ErrConflict)
	}
	if err := service.DeleteRole(ctx, owner.ID); !errors.Is(err, fault.ErrConflict) {
		t.Fatalf("delete builtin role error = %v, want %v", err, fault.ErrConflict)
	}

	reloaded, err := service.GetRole(ctx, owner.ID)
	if err != nil {
		t.Fatalf("get builtin role: %v", err)
	}
	if reloaded.Name != owner.Name {
		t.Fatalf("builtin role name = %q, want %q", reloaded.Name, owner.Name)
	}
	if got := reloaded.Permissions["users"]; got != Edit {
		t.Fatalf("builtin users permission = %q, want %q", got, Edit)
	}
}
