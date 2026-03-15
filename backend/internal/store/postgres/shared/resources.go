package pgutil

import (
	"context"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/store/db"
)

func GetGroup(ctx context.Context, queries *db.Queries, id uuid.UUID) (domain.Group, error) {
	row, err := queries.GetGroup(ctx, id)
	if err != nil {
		return domain.Group{}, err
	}

	return domain.Group{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		MemberCount: row.MemberCount,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func GetUser(ctx context.Context, queries *db.Queries, id uuid.UUID) (domain.User, error) {
	row, err := queries.GetUser(ctx, id)
	if err != nil {
		return domain.User{}, err
	}

	return MapUserRow(row)
}

func MapUserRow(row db.User) (domain.User, error) {
	source, err := ToSource(row.Source)
	if err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:          row.ID,
		UPN:         row.Upn,
		DisplayName: row.DisplayName,
		Department:  row.Department,
		Source:      source,
		Admin:       false,
		Access:      []domain.PermissionGrant{},
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}
