//go:build postgres

package directory

import (
	"strconv"
	"testing"

	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestListGroupsFiltersValuesByInternalID(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	var wantedID int64
	if err := db.QueryRow(ctx, `INSERT INTO directory_groups (source,external_id,display_name)
VALUES ('entra','external-wanted','Search Result') RETURNING id`).Scan(&wantedID); err != nil {
		t.Fatalf("insert wanted group: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO directory_groups (source,external_id,display_name)
VALUES ('entra',$1,'Search Decoy')`, strconv.FormatInt(wantedID, 10)); err != nil {
		t.Fatalf("insert decoy group: %v", err)
	}

	groups, count, err := store.ListGroups(ctx, GroupListParams{
		ListParams: listing.Params{Q: "Search"},
		Values:     []string{strconv.FormatInt(wantedID, 10)},
	})
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if count != 1 || len(groups) != 1 || groups[0].ID != wantedID {
		t.Fatalf("groups = %+v, count = %d; want only internal ID %d", groups, count, wantedID)
	}
}
