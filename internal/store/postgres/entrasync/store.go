package entrasync

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	graphsync "github.com/woodleighschool/go-entrasync"

	appentrasync "github.com/woodleighschool/woodgate/internal/app/entrasync"
	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/store/db"
	"github.com/woodleighschool/woodgate/internal/store/postgres"
)

type Store struct {
	store *postgres.Store
}

func New(store *postgres.Store) *Store {
	return &Store{store: store}
}

// ReconcileSnapshot upserts current Entra objects and removes missing Entra groups.
func (dataStore *Store) ReconcileSnapshot(
	ctx context.Context,
	snapshot *graphsync.Snapshot,
) (appentrasync.Result, error) {
	if snapshot == nil {
		snapshot = &graphsync.Snapshot{
			Users:   []graphsync.User{},
			Groups:  []graphsync.Group{},
			Members: map[uuid.UUID][]uuid.UUID{},
		}
	}

	userIDs := collectUserIDs(snapshot.Users)
	groupIDs := collectGroupIDs(snapshot.Groups)

	err := dataStore.store.RunInTx(ctx, func(queries *db.Queries) error {
		if upsertErr := upsertEntraUsers(ctx, queries, snapshot.Users); upsertErr != nil {
			return upsertErr
		}

		if upsertErr := upsertEntraGroups(ctx, queries, snapshot.Groups); upsertErr != nil {
			return upsertErr
		}

		if convertErr := queries.ConvertMissingEntraUsersToLocal(ctx, userIDs); convertErr != nil {
			return fmt.Errorf("convert missing users to local: %w", convertErr)
		}

		if deleteErr := queries.DeleteMissingGroups(ctx, groupIDs); deleteErr != nil {
			return fmt.Errorf("delete missing groups: %w", deleteErr)
		}

		if deleteErr := queries.DeleteAllGroupMemberships(ctx); deleteErr != nil {
			return fmt.Errorf("delete group memberships: %w", deleteErr)
		}

		if addErr := addUserMembers(ctx, queries, snapshot.Members); addErr != nil {
			return addErr
		}

		return nil
	})
	if err != nil {
		return appentrasync.Result{}, err
	}

	return appentrasync.Result{
		Users:       len(snapshot.Users),
		Groups:      len(snapshot.Groups),
		Memberships: countMemberships(snapshot.Members),
	}, nil
}

func upsertEntraUsers(ctx context.Context, queries *db.Queries, users []graphsync.User) error {
	for _, user := range users {
		_, err := queries.UpsertUser(ctx, db.UpsertUserParams{
			ID:          user.ID,
			Upn:         user.UPN,
			DisplayName: user.DisplayName,
			Department:  user.Department,
			Source:      string(domain.PrincipalSourceEntra),
		})
		if err != nil {
			return fmt.Errorf("upsert user %q: %w", user.ID, err)
		}
	}

	return nil
}

func upsertEntraGroups(ctx context.Context, queries *db.Queries, groups []graphsync.Group) error {
	for _, group := range groups {
		_, err := queries.UpsertGroup(ctx, db.UpsertGroupParams{
			ID:          group.ID,
			Name:        group.DisplayName,
			Description: group.Description,
		})
		if err != nil {
			return fmt.Errorf("upsert group %q: %w", group.ID, err)
		}
	}

	return nil
}

func addUserMembers(ctx context.Context, queries *db.Queries, membersByGroup map[uuid.UUID][]uuid.UUID) error {
	for groupID, memberIDs := range membersByGroup {
		for _, memberID := range memberIDs {
			membershipID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("create synced membership id: %w", err)
			}

			err = queries.AddSyncedGroupMembership(ctx, db.AddSyncedGroupMembershipParams{
				ID:      membershipID,
				GroupID: groupID,
				UserID:  memberID,
			})
			if err != nil {
				return fmt.Errorf("add member %q to group %q: %w", memberID, groupID, err)
			}
		}
	}

	return nil
}

func collectUserIDs(users []graphsync.User) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}

func collectGroupIDs(groups []graphsync.Group) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	return ids
}

func countMemberships(membersByGroup map[uuid.UUID][]uuid.UUID) int {
	total := 0
	for _, memberIDs := range membersByGroup {
		total += len(memberIDs)
	}
	return total
}
