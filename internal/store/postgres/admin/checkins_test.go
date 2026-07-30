package admin_test

import (
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/config"
	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/store/postgres"
	adminpostgres "github.com/woodleighschool/woodgate/internal/store/postgres/admin"
)

func TestListCheckins_SearchesAndFiltersResolvedFields(t *testing.T) {
	portValue := os.Getenv("WOODGATE_TEST_DATABASE_PORT")
	if portValue == "" {
		t.Skip("WOODGATE_TEST_DATABASE_PORT is not set")
	}
	port, err := strconv.Atoi(portValue)
	if err != nil {
		t.Fatalf("parse test database port: %v", err)
	}

	ctx := t.Context()
	store, err := postgres.New(ctx, config.DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     port,
		User:     "woodgate",
		Password: "woodgate",
		Name:     "woodgate_test",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Fatalf("create test store: %v", err)
	}
	t.Cleanup(store.Close)

	_, err = store.Pool().Exec(ctx, "TRUNCATE TABLE checkins, locations, users CASCADE")
	if err != nil {
		t.Fatalf("reset test data: %v", err)
	}

	matchingUserID := uuid.MustParse("01986aa4-f400-7000-8000-000000000001")
	locationID := uuid.MustParse("01986aa4-f400-7000-8000-000000000002")
	matchingCheckinID := uuid.MustParse("01986aa4-f400-7000-8000-000000000003")
	otherUserID := uuid.MustParse("01986aa4-f400-7000-8000-000000000004")
	otherCheckinID := uuid.MustParse("01986aa4-f400-7000-8000-000000000005")

	_, err = store.Pool().Exec(ctx, `
INSERT INTO users (id, upn, display_name, department, source)
VALUES
  ($1, 'matching@example.test', 'Matching User', 'Operations', 'entra'),
  ($2, 'other@example.test', 'Other User', 'Teaching', 'entra')
`, matchingUserID, otherUserID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	_, err = store.Pool().Exec(ctx, `
INSERT INTO locations (id, name)
VALUES ($1, 'Main Office')
`, locationID)
	if err != nil {
		t.Fatalf("seed location: %v", err)
	}
	_, err = store.Pool().Exec(ctx, `
INSERT INTO checkins (
  id,
  user_id,
  location_id,
  direction,
  notes,
  created_by_kind,
  created_by_id
)
VALUES
  ($1, $2, $3, 'check_in', '', 'user', $2),
  ($4, $5, $3, 'check_out', '', 'user', $5)
`, matchingCheckinID, matchingUserID, locationID, otherCheckinID, otherUserID)
	if err != nil {
		t.Fatalf("seed checkin: %v", err)
	}

	checkinStore := adminpostgres.New(store)
	items, total, err := checkinStore.ListCheckins(ctx, domain.CheckinListOptions{
		ListOptions: domain.ListOptions{Search: "Matching"},
	}, nil)
	if err != nil {
		t.Fatalf("list checkins: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(items) != 1 || items[0].ID != matchingCheckinID {
		t.Fatalf("items = %#v, want checkin %s", items, matchingCheckinID)
	}
	if items[0].UserDisplayName != "Matching User" ||
		items[0].Department != "Operations" ||
		items[0].LocationName != "Main Office" {
		t.Fatalf("resolved fields = %#v", items[0])
	}

	items, total, err = checkinStore.ListCheckins(ctx, domain.CheckinListOptions{
		Department: "Teaching",
	}, nil)
	if err != nil {
		t.Fatalf("filter checkins by department: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != otherCheckinID {
		t.Fatalf("department-filtered items = %#v, total = %d", items, total)
	}

	departments, err := checkinStore.ListCheckinDepartments(ctx, nil)
	if err != nil {
		t.Fatalf("list checkin departments: %v", err)
	}
	if !slices.Equal(departments, []string{"Operations", "Teaching"}) {
		t.Fatalf("departments = %#v", departments)
	}
}
