//go:build postgres

package directory

import (
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/goodies/auth/authn"

	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestProviderSnapshotKeepsLocalAndEntraUsersSeparate(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := NewUserService(store)
	authnStore := NewAuthnStore(store)
	var adminID int64
	if err := database.QueryRow(ctx, `SELECT id FROM authz_roles WHERE key = 'admin'`).Scan(&adminID); err != nil {
		t.Fatalf("get admin role: %v", err)
	}

	local, err := service.Create(ctx, UserCreate{
		Email:    "shared@example.invalid",
		Name:     "Local User",
		RoleIDs:  []int64{adminID},
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create local user: %v", err)
	}
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		Users: []ProviderUser{{
			ExternalID:        "provider-identity",
			Mail:              local.Email,
			UserPrincipalName: "provider-upn@example.invalid",
			DisplayName:       "Provider Identity",
			Enabled:           true,
		}},
	}); err != nil {
		t.Fatalf("apply provider snapshot: %v", err)
	}

	var providerID int64
	if err := database.QueryRow(ctx, `
SELECT id FROM users
WHERE source = 'entra' AND external_id = 'provider-identity'`).Scan(&providerID); err != nil {
		t.Fatalf("load provider user: %v", err)
	}
	if providerID == local.ID {
		t.Fatalf("provider user reused local user ID %d", local.ID)
	}
	if principal, err := authnStore.GetSSOPrincipalByEmail(ctx, local.Email); err != nil || principal.ID != providerID {
		t.Fatalf("SSO lookup before application grant = %+v, %v", principal, err)
	}

	granted, err := service.SetRolesByEmail(ctx, local.Email, []int64{adminID})
	if err != nil {
		t.Fatalf("grant provider role by email: %v", err)
	}
	if granted.ID != providerID {
		t.Fatalf("granted user ID = %d, want provider user %d", granted.ID, providerID)
	}
	if sso, err := authnStore.GetSSOPrincipalByEmail(ctx, local.Email); err != nil || sso.ID != providerID {
		t.Fatalf("SSO user after provider grant = %+v, %v", sso, err)
	}
	if login, err := authnStore.GetPasswordIdentityByEmail(ctx, local.Email); err != nil || login.ID != local.ID {
		t.Fatalf("password login user = %+v, %v", login, err)
	}
}

func TestSSOLookupDoesNotUseUPNAsAlternateAccountIdentifier(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	authnStore := NewAuthnStore(store)
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		Users: []ProviderUser{{
			ExternalID:        "provider-user",
			Mail:              "canonical@example.invalid",
			UserPrincipalName: "alternate@example.invalid",
			DisplayName:       "Provider User",
			Enabled:           true,
		}},
	}); err != nil {
		t.Fatalf("apply provider snapshot: %v", err)
	}
	if _, err := authnStore.GetSSOPrincipalByEmail(ctx, "alternate@example.invalid"); !errors.Is(err, authn.ErrPrincipalNotFound) {
		t.Fatalf("SSO lookup by UPN error = %v, want %v", err, authn.ErrPrincipalNotFound)
	}
	user, err := authnStore.GetSSOPrincipalByEmail(ctx, "canonical@example.invalid")
	if err != nil {
		t.Fatalf("SSO lookup by canonical email: %v", err)
	}
	if user.Email != "canonical@example.invalid" {
		t.Fatalf("SSO user email = %q, want canonical email", user.Email)
	}
}

func TestApplyProviderSnapshotReconcilesUsersAndGroups(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)

	first := ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []ProviderGroup{
			{ExternalID: "g-eng", DisplayName: "Engineering"},
			{ExternalID: "g-ops", DisplayName: "Operations"},
		},
		Users: []ProviderUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.invalid",
				DisplayName:       "Alice",
				Department:        "Engineering",
				Enabled:           true,
				GroupExternalIDs:  []string{"g-eng", "g-ops"},
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.invalid",
				DisplayName:       "Bob",
				Department:        "Operations",
				Enabled:           true,
				GroupExternalIDs:  []string{"g-ops"},
			},
		},
	}
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, first); err != nil {
		t.Fatalf("apply first snapshot: %v", err)
	}

	second := ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Groups:      []ProviderGroup{{ExternalID: "g-ops", DisplayName: "Operations"}},
		Users: []ProviderUser{{
			ExternalID:        "u-alice",
			UserPrincipalName: "alice@example.invalid",
			DisplayName:       "Alice Updated",
			Department:        "Operations",
			Enabled:           true,
			GroupExternalIDs:  []string{"g-ops"},
		}},
	}
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, second); err != nil {
		t.Fatalf("apply second snapshot: %v", err)
	}

	var name, department string
	if err := database.QueryRow(ctx, `
SELECT name, COALESCE(department, '') FROM users
WHERE source = 'entra' AND external_id = 'u-alice'`).Scan(&name, &department); err != nil {
		t.Fatalf("load Alice: %v", err)
	}
	if name != "Alice Updated" || department != "Operations" {
		t.Fatalf("Alice name/department = %q/%q, want updated Operations", name, department)
	}
	var bobDeletedAt *time.Time
	if err := database.QueryRow(ctx, `
SELECT deleted_at FROM users
WHERE source = 'entra' AND external_id = 'u-bob'`).Scan(&bobDeletedAt); err != nil {
		t.Fatalf("load Bob: %v", err)
	}
	if bobDeletedAt == nil {
		t.Fatal("Bob remains active after leaving the snapshot")
	}
	var bobMemberships int
	if err := database.QueryRow(ctx, `
SELECT count(*)
FROM directory_group_memberships membership
JOIN users ON users.id = membership.user_id
WHERE users.source = 'entra' AND users.external_id = 'u-bob'`).Scan(&bobMemberships); err != nil {
		t.Fatalf("count Bob memberships: %v", err)
	}
	if bobMemberships != 0 {
		t.Fatalf("Bob membership count = %d, want 0", bobMemberships)
	}
}

func TestApplyProviderSnapshotKeepsReplacedEntraObjectsDistinct(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := NewUserService(store)
	authnStore := NewAuthnStore(store)
	var adminID int64
	if err := database.QueryRow(ctx, `SELECT id FROM authz_roles WHERE key = 'admin'`).Scan(&adminID); err != nil {
		t.Fatalf("get admin role: %v", err)
	}

	user := ProviderUser{
		ExternalID:        "old-object-id",
		UserPrincipalName: "recreated@example.invalid",
		Mail:              "recreated@example.invalid",
		DisplayName:       "Recreated User",
		Enabled:           true,
	}
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{Users: []ProviderUser{user}}); err != nil {
		t.Fatalf("apply original user snapshot: %v", err)
	}
	old, err := service.SetRolesByEmail(ctx, user.Mail, []int64{adminID})
	if err != nil {
		t.Fatalf("grant original user role: %v", err)
	}

	user.ExternalID = "new-object-id"
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{Users: []ProviderUser{user}}); err != nil {
		t.Fatalf("apply replacement user snapshot: %v", err)
	}

	var oldDeletedAt *time.Time
	if err := database.QueryRow(ctx, `SELECT deleted_at FROM users WHERE id = $1`, old.ID).Scan(&oldDeletedAt); err != nil {
		t.Fatalf("load original user: %v", err)
	}
	if oldDeletedAt == nil {
		t.Fatal("original user remains active after its object left the snapshot")
	}
	var newUserID int64
	if err := database.QueryRow(ctx, `
SELECT id FROM users
WHERE source = 'entra' AND external_id = 'new-object-id'`).Scan(&newUserID); err != nil {
		t.Fatalf("load replacement user: %v", err)
	}
	if newUserID == old.ID {
		t.Fatalf("replacement user reused original user ID %d", old.ID)
	}
	replacement, err := service.Get(ctx, newUserID)
	if err != nil {
		t.Fatalf("load replacement user: %v", err)
	}
	if len(replacement.EffectiveRoles) != 0 {
		t.Fatalf("replacement effective roles = %+v, want none", replacement.EffectiveRoles)
	}
	if principal, err := authnStore.GetSSOPrincipalByEmail(ctx, user.Mail); err != nil || principal.ID != newUserID {
		t.Fatalf("replacement authentication principal = %+v, error = %v", principal, err)
	}
}
