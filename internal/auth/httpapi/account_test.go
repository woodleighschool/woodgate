package httpapi

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodgate/internal/authorization"
	"github.com/woodleighschool/woodgate/internal/directory"
)

func TestNewAccountOutputIncludesEffectivePermissions(t *testing.T) {
	t.Parallel()

	permissions := map[authorization.Resource]authorization.Access{
		"users": authorization.Edit,
	}
	output, err := newAccountOutput(
		t.Context(),
		&directory.Account{User: directory.User{ID: 42}},
		staticPermissionDirectory{permissions: permissions},
	)
	if err != nil {
		t.Fatalf("newAccountOutput() error = %v", err)
	}
	if output.Body.User.ID != 42 {
		t.Fatalf("user ID = %d, want 42", output.Body.User.ID)
	}
	if got := output.Body.EffectivePermissions["users"]; got != authorization.Edit {
		t.Fatalf("users permission = %q, want %q", got, authorization.Edit)
	}
}

type staticPermissionDirectory struct {
	permissions map[authorization.Resource]authorization.Access
}

func (directory staticPermissionDirectory) EffectivePermissions(
	context.Context,
	int64,
) (map[authorization.Resource]authorization.Access, error) {
	return directory.permissions, nil
}
