package authorization

import "context"

type Service struct {
	store *Store
}

func NewService(store *Store) *Service { return &Service{store: store} }

func (service *Service) Can(
	ctx context.Context,
	userID int64,
	resource Resource,
	required Access,
) (bool, error) {
	permissions, err := service.store.EffectivePermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	return permissions[resource].level() >= required.level(), nil
}

func (service *Service) EffectivePermissions(ctx context.Context, userID int64) (map[Resource]Access, error) {
	return service.store.EffectivePermissions(ctx, userID)
}

func (service *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return service.store.ListRoles(ctx)
}

func (service *Service) GetRole(ctx context.Context, id int64) (*Role, error) {
	return service.store.GetRole(ctx, id)
}

func (service *Service) CreateRole(ctx context.Context, mutation RoleMutation) (*Role, error) {
	return service.store.CreateRole(ctx, mutation)
}

func (service *Service) UpdateRole(ctx context.Context, id int64, mutation RoleMutation) (*Role, error) {
	return service.store.UpdateRole(ctx, id, mutation)
}

func (service *Service) DeleteRole(ctx context.Context, id int64) error {
	return service.store.DeleteRole(ctx, id)
}

func (service *Service) ListAssignments(ctx context.Context) ([]Assignment, error) {
	return service.store.ListAssignments(ctx)
}

func (service *Service) ReplaceAssignments(ctx context.Context, mutation AssignmentMutation) error {
	return service.store.ReplaceAssignments(ctx, mutation)
}

func (service *Service) AssignOwner(ctx context.Context, userID int64) error {
	return service.store.AssignOwner(ctx, userID)
}
