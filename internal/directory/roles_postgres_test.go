//go:build postgres

package directory

import (
	"context"
	"strconv"
	"testing"

	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestUserAndGroupResourcesOwnRoleMembership(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	users := NewUserService(store)
	authnStore := NewAuthnStore(store)

	var adminID int64
	if err := database.QueryRow(ctx, `SELECT id FROM authz_roles WHERE key = 'admin'`).Scan(&adminID); err != nil {
		t.Fatalf("get admin role: %v", err)
	}
	admin, err := users.Create(ctx, UserCreate{
		Email: "admin@example.invalid", Password: "correct horse battery staple", RoleIDs: []int64{adminID},
	})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if len(admin.Roles) != 1 || admin.Roles[0].Key != "admin" || len(admin.EffectiveRoles) != 1 {
		t.Fatalf("admin roles = direct %+v effective %+v", admin.Roles, admin.EffectiveRoles)
	}
	if _, err := authnStore.GetPasswordIdentityByEmail(ctx, admin.Email); err != nil {
		t.Fatalf("get admin login: %v", err)
	}

	noAccess, err := users.Create(ctx, UserCreate{
		Email: "no-access@example.invalid", Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("create no-access user: %v", err)
	}
	if len(noAccess.EffectiveRoles) != 0 {
		t.Fatalf("no-access effective roles = %+v", noAccess.EffectiveRoles)
	}
	if principal, err := authnStore.GetPasswordIdentityByEmail(ctx, noAccess.Email); err != nil || principal.ID != noAccess.ID {
		t.Fatalf("no-access authentication principal = %+v, error = %v", principal, err)
	}

	var groupID, roleID int64
	if err := database.QueryRow(ctx, `
INSERT INTO directory_groups (source, external_id, display_name)
VALUES ('entra', 'role-group', 'Role Group') RETURNING id`).Scan(&groupID); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := database.QueryRow(ctx, `
INSERT INTO authz_roles (key, name) VALUES ('group-viewer', 'Group Viewer') RETURNING id`).Scan(&roleID); err != nil {
		t.Fatalf("create role: %v", err)
	}
	if _, err := database.Exec(ctx, `
INSERT INTO directory_group_memberships (user_id, group_id) VALUES ($1, $2)`, noAccess.ID, groupID); err != nil {
		t.Fatalf("create membership: %v", err)
	}
	group, err := store.UpdateGroup(ctx, groupID, GroupMutation{RoleIDs: []int64{roleID}})
	if err != nil {
		t.Fatalf("update group: %v", err)
	}
	if len(group.Roles) != 1 || group.Roles[0].Name != "Group Viewer" {
		t.Fatalf("group roles = %+v", group.Roles)
	}
	assertRoleSummaries(t, ctx, users, roleID)
	inherited, err := users.Get(ctx, noAccess.ID)
	if err != nil {
		t.Fatalf("get inherited user: %v", err)
	}
	if len(inherited.Roles) != 0 || len(inherited.EffectiveRoles) != 1 || inherited.EffectiveRoles[0].ID != roleID {
		t.Fatalf("inherited roles = direct %+v effective %+v", inherited.Roles, inherited.EffectiveRoles)
	}

	withRole, count, err := users.List(ctx, UserListParams{
		ListParams: listing.Params{PageSize: 20}, RoleValues: []string{strconv.FormatInt(roleID, 10)},
	})
	if err != nil || count != 1 || len(withRole) != 1 || withRole[0].ID != noAccess.ID {
		t.Fatalf("role-filtered users = %+v count %d error %v", withRole, count, err)
	}
	withoutRole, count, err := users.List(ctx, UserListParams{
		ListParams: listing.Params{PageSize: 20}, RoleValues: []string{"none"},
	})
	if err != nil || count != 0 || len(withoutRole) != 0 {
		t.Fatalf("no-access users = %+v count %d error %v", withoutRole, count, err)
	}
}

func assertRoleSummaries(t *testing.T, ctx context.Context, users *UserService, roleID int64) {
	t.Helper()
	roles, err := users.ListRoleSummaries(ctx)
	if err != nil {
		t.Fatalf("list role summaries: %v", err)
	}
	if len(roles) != 2 || roles[0].Key != "admin" || roles[1].ID != roleID {
		t.Fatalf("role summaries = %+v", roles)
	}
}
