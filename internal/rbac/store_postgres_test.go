//go:build postgres

package rbac

import (
	"errors"
	"testing"

	"github.com/woodleighschool/goodies/auth/authz"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestEffectivePermissionsIncludesGroupRoles(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service, err := authz.NewService(store, Resources())
	if err != nil {
		t.Fatal(err)
	}

	var userID int64
	if err := database.QueryRow(ctx, `
INSERT INTO users (email, name, source)
VALUES ('group-member@example.invalid', 'Group Member', 'local')
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

	role, err := store.CreateRole(ctx, RoleMutation{
		Key:  "group-users-viewer",
		Name: "Group users viewer",
		Permissions: map[authz.Resource]authz.Access{
			"users": authz.View,
		},
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if _, err := database.Exec(ctx, `INSERT INTO authz_group_roles (group_id, role_id) VALUES ($1, $2)`, groupID, role.ID); err != nil {
		t.Fatalf("assign group role: %v", err)
	}

	permissions, err := service.EffectivePermissions(ctx, userID)
	if err != nil {
		t.Fatalf("get effective permissions: %v", err)
	}
	if got := permissions["users"]; got != authz.View {
		t.Fatalf("users permission = %q, want %q", got, authz.View)
	}
	if got := permissions["groups"]; got != authz.None {
		t.Fatalf("groups permission = %q, want %q", got, authz.None)
	}
	if allowed, err := service.Can(ctx, userID, "users", authz.Edit); err != nil {
		t.Fatalf("check edit access: %v", err)
	} else if allowed {
		t.Fatal("edit access allowed, want denied")
	}
	if allowed, err := service.CanAll(ctx, userID,
		authz.Requirement{Resource: ResourceUsers, Access: authz.View},
		authz.Requirement{Resource: ResourceGroups, Access: authz.View},
	); err != nil {
		t.Fatalf("check combined access: %v", err)
	} else if allowed {
		t.Fatal("combined access allowed without groups permission")
	}
	if allowed, err := service.CanAll(ctx, userID,
		authz.Requirement{Resource: ResourceUsers, Access: authz.View},
	); err != nil {
		t.Fatalf("check allowed combined access: %v", err)
	} else if !allowed {
		t.Fatal("combined access denied with required permission")
	}
	direct, err := store.CreateRole(ctx, RoleMutation{
		Key: "direct-users-editor", Name: "Direct users editor",
		Permissions: map[authz.Resource]authz.Access{ResourceUsers: authz.Edit},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, `INSERT INTO authz_user_roles (user_id, role_id) VALUES ($1, $2)`, userID, direct.ID); err != nil {
		t.Fatal(err)
	}
	grants, err := store.Grants(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	levels := map[authz.Access]int{}
	for _, grant := range grants {
		if grant.Resource == ResourceUsers {
			levels[grant.Access]++
		}
	}
	if levels[authz.View] != 1 || levels[authz.Edit] != 1 {
		t.Fatalf("store aggregated direct and inherited grants: %v", levels)
	}
	permissions, err = service.EffectivePermissions(ctx, userID)
	if err != nil || permissions[ResourceUsers] != authz.Edit {
		t.Fatalf("merged permissions = %v, error = %v", permissions, err)
	}
}

func TestBuiltinRoleCannotBeChangedOrDeleted(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)

	roles, err := store.ListRoles(ctx)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	var admin *Role
	for i := range roles {
		if roles[i].Key == "admin" {
			admin = &roles[i]
			break
		}
	}
	if admin == nil {
		t.Fatal("admin role not found")
	}

	_, err = store.UpdateRole(ctx, admin.ID, RoleMutation{
		Name: "Changed admin",
		Permissions: map[authz.Resource]authz.Access{
			"users": authz.View,
		},
	})
	if !errors.Is(err, fault.ErrConflict) {
		t.Fatalf("update builtin role error = %v, want %v", err, fault.ErrConflict)
	}
	if err := store.DeleteRole(ctx, admin.ID); !errors.Is(err, fault.ErrConflict) {
		t.Fatalf("delete builtin role error = %v, want %v", err, fault.ErrConflict)
	}

	reloaded, err := store.GetRole(ctx, admin.ID)
	if err != nil {
		t.Fatalf("get builtin role: %v", err)
	}
	if reloaded.Name != admin.Name {
		t.Fatalf("builtin role name = %q, want %q", reloaded.Name, admin.Name)
	}
	if got := reloaded.Permissions["users"]; got != authz.Edit {
		t.Fatalf("builtin users permission = %q, want %q", got, authz.Edit)
	}
}
