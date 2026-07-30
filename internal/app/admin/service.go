package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
	adminpostgres "github.com/woodleighschool/woodgate/internal/store/postgres/admin"
)

type Service struct {
	store      *adminpostgres.Store
	authorizer *authz.CasbinAuthorizer
	mediaRoot  string
}

func New(store *adminpostgres.Store, authorizer *authz.CasbinAuthorizer, mediaRoot string) (*Service, error) {
	mkdirErr := ensureMediaRoot(mediaRoot)
	if mkdirErr != nil {
		return nil, mkdirErr
	}

	return &Service{
		store:      store,
		authorizer: authorizer,
		mediaRoot:  mediaRoot,
	}, nil
}

func (service *Service) ListUsers(ctx context.Context, options domain.UserListOptions) ([]domain.User, int32, error) {
	return service.store.ListUsers(ctx, options)
}

func (service *Service) GetUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return service.store.GetUser(ctx, id)
}

func (service *Service) UpdateUserAccess(
	ctx context.Context,
	id uuid.UUID,
	admin bool,
	access []domain.PermissionGrant,
) (domain.User, error) {
	item, err := service.store.UpdateUserAccess(ctx, id, admin, access)
	if err != nil {
		return domain.User{}, err
	}
	reloadErr := service.reloadAuthorizer(ctx)
	if reloadErr != nil {
		return domain.User{}, reloadErr
	}
	return item, nil
}

func (service *Service) ListGroups(
	ctx context.Context,
	options domain.GroupListOptions,
) ([]domain.Group, int32, error) {
	return service.store.ListGroups(ctx, options)
}

func (service *Service) GetGroup(ctx context.Context, id uuid.UUID) (domain.Group, error) {
	return service.store.GetGroup(ctx, id)
}

func (service *Service) ListGroupMemberships(
	ctx context.Context,
	options domain.GroupMembershipListOptions,
) ([]domain.GroupMembership, int32, error) {
	return service.store.ListGroupMemberships(ctx, options)
}

func (service *Service) GetGroupMembership(ctx context.Context, id uuid.UUID) (domain.GroupMembership, error) {
	return service.store.GetGroupMembership(ctx, id)
}

func (service *Service) ListAssets(
	ctx context.Context,
	options domain.AssetListOptions,
) ([]domain.Asset, int32, error) {
	return service.store.ListAssets(ctx, options)
}

func (service *Service) GetAsset(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	return service.store.GetAsset(ctx, id)
}

func (service *Service) ListLocations(
	ctx context.Context,
	options domain.LocationListOptions,
) ([]domain.Location, int32, error) {
	return service.store.ListLocations(ctx, options)
}

func (service *Service) CreateLocation(
	ctx context.Context,
	name string,
	description string,
	enabled bool,
	notes bool,
	photo bool,
	backgroundAssetID *uuid.UUID,
	logoAssetID *uuid.UUID,
	groupIDs []uuid.UUID,
) (domain.Location, error) {
	if validateErr := service.validateLocationAssetRefs(ctx, backgroundAssetID, logoAssetID); validateErr != nil {
		return domain.Location{}, validateErr
	}

	return service.store.CreateLocation(
		ctx,
		name,
		description,
		enabled,
		notes,
		photo,
		backgroundAssetID,
		logoAssetID,
		groupIDs,
	)
}

func (service *Service) GetLocation(ctx context.Context, id uuid.UUID) (domain.Location, error) {
	return service.store.GetLocation(ctx, id)
}

func (service *Service) UpdateLocation(
	ctx context.Context,
	id uuid.UUID,
	name string,
	description string,
	enabled bool,
	notes bool,
	photo bool,
	backgroundAssetID *uuid.UUID,
	logoAssetID *uuid.UUID,
	groupIDs []uuid.UUID,
) (domain.Location, error) {
	if validateErr := service.validateLocationAssetRefs(ctx, backgroundAssetID, logoAssetID); validateErr != nil {
		return domain.Location{}, validateErr
	}

	return service.store.UpdateLocation(
		ctx,
		id,
		name,
		description,
		enabled,
		notes,
		photo,
		backgroundAssetID,
		logoAssetID,
		groupIDs,
	)
}

func (service *Service) DeleteLocation(ctx context.Context, id uuid.UUID) error {
	return service.store.DeleteLocation(ctx, id)
}

func (service *Service) ListCheckins(
	ctx context.Context,
	options domain.CheckinListOptions,
	allowedLocationIDs []uuid.UUID,
) ([]domain.Checkin, int32, error) {
	return service.store.ListCheckins(ctx, options, allowedLocationIDs)
}

func (service *Service) CreateCheckin(
	ctx context.Context,
	userID uuid.UUID,
	locationID uuid.UUID,
	direction domain.CheckinDirection,
	notes string,
	photoContent []byte,
	createdByKind domain.PermissionSubjectKind,
	createdByID uuid.UUID,
) (domain.Checkin, error) {
	var assetID *uuid.UUID
	if len(photoContent) > 0 {
		asset, err := service.createStoredAsset(ctx, nil, domain.AssetTypePhoto, photoContent)
		if err != nil {
			return domain.Checkin{}, err
		}
		assetID = &asset.ID
	}

	item, err := service.store.CreateCheckin(
		ctx,
		userID,
		locationID,
		direction,
		notes,
		assetID,
		createdByKind,
		createdByID,
	)
	if err != nil {
		if assetID != nil {
			_ = service.DeleteAsset(ctx, *assetID)
		}
		return domain.Checkin{}, err
	}

	return item, nil
}

func (service *Service) GetCheckin(
	ctx context.Context,
	id uuid.UUID,
	allowedLocationIDs []uuid.UUID,
) (domain.Checkin, error) {
	return service.store.GetCheckin(ctx, id, allowedLocationIDs)
}

func (service *Service) ListAPIKeys(
	ctx context.Context,
	options domain.APIKeyListOptions,
) ([]domain.APIKey, int32, error) {
	return service.store.ListAPIKeys(ctx, options)
}

func (service *Service) CreateAPIKey(
	ctx context.Context,
	name string,
	prefix string,
	hash string,
	expiresAt *time.Time,
) (domain.APIKey, error) {
	return service.store.CreateAPIKey(ctx, name, prefix, hash, expiresAt)
}

func (service *Service) GetAPIKey(ctx context.Context, id uuid.UUID) (domain.APIKey, error) {
	return service.store.GetAPIKey(ctx, id)
}

func (service *Service) UpdateAPIKeyAccess(
	ctx context.Context,
	id uuid.UUID,
	admin bool,
	access []domain.PermissionGrant,
) (domain.APIKey, error) {
	item, err := service.store.UpdateAPIKeyAccess(ctx, id, admin, access)
	if err != nil {
		return domain.APIKey{}, err
	}
	reloadErr := service.reloadAuthorizer(ctx)
	if reloadErr != nil {
		return domain.APIKey{}, reloadErr
	}
	return item, nil
}

func (service *Service) DeleteAPIKey(ctx context.Context, id uuid.UUID) error {
	return service.store.DeleteAPIKey(ctx, id)
}

func (service *Service) reloadAuthorizer(ctx context.Context) error {
	if service.authorizer == nil {
		return nil
	}
	reloadErr := service.authorizer.Reload(ctx)
	if reloadErr != nil {
		return fmt.Errorf("reload authorizer: %w", reloadErr)
	}
	return nil
}

func (service *Service) validateLocationAssetRefs(
	ctx context.Context,
	backgroundAssetID *uuid.UUID,
	logoAssetID *uuid.UUID,
) error {
	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Location is invalid."}

	validate := func(field string, id *uuid.UUID) {
		if id == nil {
			return
		}

		asset, err := service.store.GetAsset(ctx, *id)
		if err != nil {
			validationErr.Add(field, "must reference an existing asset", "invalid")
			return
		}
		if asset.Type != domain.AssetTypeAsset {
			validationErr.Add(field, "must reference a reusable asset", "invalid")
		}
	}

	validate("background_asset_id", backgroundAssetID)
	validate("logo_asset_id", logoAssetID)

	if validationErr.HasFieldErrors() {
		return validationErr
	}

	return nil
}
