package directory

import (
	"context"
	"strings"
)

// UserService owns user management and application access.
type UserService struct {
	store *Store
}

// NewUserService returns the user-management service.
func NewUserService(store *Store) *UserService {
	return &UserService{store: store}
}

func (s *UserService) Get(ctx context.Context, id int64) (*User, error) {
	return s.store.GetUserByID(ctx, id)
}

func (s *UserService) List(ctx context.Context, params UserListParams) ([]User, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	return s.store.ListUsers(ctx, params)
}

// ListRoleSummaries returns role identities usable within user read surfaces.
func (s *UserService) ListRoleSummaries(ctx context.Context) ([]RoleSummary, error) {
	return s.store.ListRoleSummaries(ctx)
}

func (s *UserService) Create(ctx context.Context, params UserCreate) (*User, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, err
	}
	hash, err := hashPassword(params.Password)
	if err != nil {
		return nil, err
	}
	return s.store.createUser(ctx, userCreateRecord{
		Email:        params.Email,
		Name:         params.Name,
		PasswordHash: hash,
		RoleIDs:      params.RoleIDs,
	})
}

// Update writes the full target record.
func (s *UserService) Update(ctx context.Context, targetID int64, params UserMutation) (*User, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, err
	}
	passwordHash, err := hashOptionalPassword(params.Password)
	if err != nil {
		return nil, err
	}
	return s.store.updateUser(ctx, targetID, userUpdateRecord{
		Name:         params.Name,
		PasswordHash: passwordHash,
		RoleIDs:      params.RoleIDs,
	})
}

// Delete hard-deletes local users and soft-deletes source-owned identities.
func (s *UserService) Delete(ctx context.Context, targetID int64) error {
	return s.store.deleteUser(ctx, targetID)
}

// SetPasswordByEmail replaces a local user's password.
func (s *UserService) SetPasswordByEmail(
	ctx context.Context,
	email string,
	password string,
) (*User, error) {
	email = strings.TrimSpace(email)
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	return s.store.setLocalUserPasswordByEmail(ctx, email, hash)
}

// SetRolesByEmail replaces a persisted user's direct roles.
func (s *UserService) SetRolesByEmail(
	ctx context.Context,
	email string,
	roleIDs []int64,
) (*User, error) {
	email = strings.TrimSpace(email)
	roleIDs = normalizeRoleIDs(roleIDs)
	if err := validateRoleIDs(roleIDs); err != nil {
		return nil, err
	}
	return s.store.setUserRolesByEmail(ctx, email, roleIDs)
}

func hashOptionalPassword(password *string) (*string, error) {
	if password == nil {
		return nil, nil
	}
	hash, err := hashPassword(*password)
	if err != nil {
		return nil, err
	}
	return &hash, nil
}
