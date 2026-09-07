//go:build postgres

package main

import (
	"io"
	"slices"
	"testing"

	"github.com/woodleighschool/goodies/auth/authz"

	"github.com/woodleighschool/woodgate/internal/rbac"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestCreateUserCommandAssignsRequestedRoles(t *testing.T) {
	database, ctx := testdb.Open(t)
	if _, err := rbac.NewStore(database).CreateRole(ctx, rbac.RoleMutation{
		Key: "operator", Name: "Operator", Permissions: map[authz.Resource]authz.Access{rbac.ResourceCheckins: authz.Edit},
	}); err != nil {
		t.Fatalf("create role: %v", err)
	}
	for _, tc := range []struct {
		name  string
		flags []string
		roles []string
	}{
		{name: "default", roles: []string{"admin"}},
		{name: "custom", flags: []string{"--role", "operator"}, roles: []string{"operator"}},
		{name: "multiple", flags: []string{"--role", "operator", "--role", "admin"}, roles: []string{"admin", "operator"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			email := tc.name + "@example.invalid"
			cmd := userCommand()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			args := []string{"--database-url", database.Config().ConnString(), "create", "--email", email, "--password", "correct-password"}
			cmd.SetArgs(append(args, tc.flags...))
			if err := cmd.ExecuteContext(ctx); err != nil {
				t.Fatalf("create user command: %v", err)
			}
			var got []string
			if err := database.QueryRow(ctx, `
SELECT array_agg(role.key ORDER BY role.key)
FROM authz_user_roles membership
JOIN authz_roles role ON role.id = membership.role_id
JOIN users ON users.id = membership.user_id
WHERE users.email = $1`, email).Scan(&got); err != nil {
				t.Fatalf("get assigned roles: %v", err)
			}
			if !slices.Equal(got, tc.roles) {
				t.Fatalf("assigned roles = %v, want %v", got, tc.roles)
			}
		})
	}
}
