package checkin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/postgres"
	"github.com/woodleighschool/woodgate/internal/station"
)

// Store persists the check-in domain in PostgreSQL.
type Store struct {
	pool    *pgxpool.Pool
	objects *bloby.Service
}

// NewStore returns a PostgreSQL-backed check-in store.
func NewStore(pool *pgxpool.Pool, objects *bloby.Service) *Store {
	return &Store{pool: pool, objects: objects}
}

type locationRow struct {
	ID                  int64     `db:"id"`
	Name                string    `db:"name"`
	Description         string    `db:"description"`
	Enabled             bool      `db:"enabled"`
	Notes               bool      `db:"notes"`
	Photo               bool      `db:"photo"`
	BackgroundObjectID  *int64    `db:"background_object_id"`
	BackgroundFilename  *string   `db:"background_filename"`
	BackgroundType      *string   `db:"background_content_type"`
	BackgroundSizeBytes *int64    `db:"background_size_bytes"`
	BackgroundSHA256    *string   `db:"background_sha256"`
	LogoObjectID        *int64    `db:"logo_object_id"`
	LogoFilename        *string   `db:"logo_filename"`
	LogoType            *string   `db:"logo_content_type"`
	LogoSizeBytes       *int64    `db:"logo_size_bytes"`
	LogoSHA256          *string   `db:"logo_sha256"`
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

const locationSelectSQL = `SELECT l.id, l.name, l.description, l.enabled, l.notes, l.photo,
    l.background_object_id,
    background.filename AS background_filename,
    background.content_type AS background_content_type,
    background.size_bytes AS background_size_bytes,
    background.sha256 AS background_sha256,
    l.logo_object_id,
    logo.filename AS logo_filename,
    logo.content_type AS logo_content_type,
    logo.size_bytes AS logo_size_bytes,
    logo.sha256 AS logo_sha256,
    l.created_at, l.updated_at
FROM locations l
LEFT JOIN storage_objects background ON background.id = l.background_object_id
LEFT JOIN storage_objects logo ON logo.id = l.logo_object_id`

func (s *Store) ListLocations(ctx context.Context, params LocationListParams) ([]Location, int, error) {
	params.ListParams = listingNormalize(params.ListParams)
	var where postgres.WhereBuilder
	if params.ListParams.Q != "" {
		where.Addf("(l.name ILIKE '%%' || %s || '%%' OR l.description ILIKE '%%' || %s || '%%')", params.ListParams.Q, params.ListParams.Q)
	}
	if params.Enabled != nil {
		where.Addf("l.enabled = %s", *params.Enabled)
	}
	whereSQL, args := where.Build()
	rows, count, err := postgres.ListWithCount[locationRow](ctx, s.pool, postgres.ListQuery{
		SelectSQL: locationSelectSQL,
		WhereSQL:  whereSQL,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			"id": {SQL: "l.id"}, "name": {SQL: "lower(l.name)"}, "enabled": {SQL: "l.enabled"},
			"created_at": {SQL: "l.created_at"}, "updated_at": {SQL: "l.updated_at"},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: "lower(l.name)"}, {SQL: "l.id"}},
		Params:       params.ListParams,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]Location, len(rows))
	for i, row := range rows {
		items[i] = locationFromRow(row)
	}
	if err := attachLocationGroups(ctx, s.pool, items); err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

func (s *Store) ListLocationGroupChoices(
	ctx context.Context,
	params listing.Params,
) ([]directory.GroupSummary, int, error) {
	var where postgres.WhereBuilder
	if params.Q != "" {
		where.Addf(`(
            group_row.display_name ILIKE '%%' || %s || '%%'
            OR group_row.mail_nickname ILIKE '%%' || %s || '%%'
        )`, params.Q, params.Q)
	}
	whereSQL, args := where.Build()
	return postgres.ListWithCount[directory.GroupSummary](ctx, s.pool, postgres.ListQuery{
		SelectSQL: `SELECT group_row.id, group_row.source::text AS source,
            group_row.display_name, COALESCE(group_row.mail_nickname, '') AS mail_nickname
            FROM directory_groups AS group_row`,
		WhereSQL: whereSQL,
		Args:     args,
		OrderKeys: map[string]postgres.OrderExpr{
			"display_name":  {SQL: "lower(group_row.display_name)"},
			"mail_nickname": {SQL: "lower(group_row.mail_nickname)", NullOrder: postgres.NullsLast},
			"source":        {SQL: "group_row.source"},
		},
		DefaultOrder: []postgres.OrderExpr{
			{SQL: "lower(group_row.display_name)"},
			{SQL: "group_row.id"},
		},
		Params: params,
	})
}

func (s *Store) ListStationLocationChoices(
	ctx context.Context,
	params listing.Params,
) ([]station.Location, int, error) {
	var where postgres.WhereBuilder
	if params.Q != "" {
		where.Addf("name ILIKE '%%' || %s || '%%'", params.Q)
	}
	whereSQL, args := where.Build()
	return postgres.ListWithCount[station.Location](ctx, s.pool, postgres.ListQuery{
		SelectSQL: "SELECT id, name FROM locations",
		WhereSQL:  whereSQL,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			"name": {SQL: "lower(name)"},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: "lower(name)"}, {SQL: "id"}},
		Params:       params,
	})
}

func (s *Store) GetLocation(ctx context.Context, id int64) (*Location, error) {
	row, err := postgres.GetOne[locationRow](ctx, s.pool, locationSelectSQL+"\nWHERE l.id = $1", id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	locations := []Location{locationFromRow(row)}
	if err := attachLocationGroups(ctx, s.pool, locations); err != nil {
		return nil, err
	}
	return &locations[0], nil
}

func (s *Store) CreateLocation(ctx context.Context, mutation LocationMutation) (*Location, error) {
	if err := s.validateLocationAttachments(ctx, mutation.BackgroundObjectID, mutation.LogoObjectID); err != nil {
		return nil, err
	}
	var id int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO locations
            (name, description, enabled, notes, photo, background_object_id, logo_object_id)
            VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, mutation.Name, mutation.Description,
			mutation.Enabled, mutation.Notes, mutation.Photo, mutation.BackgroundObjectID, mutation.LogoObjectID).Scan(&id); err != nil {
			return postgres.MutationError(err)
		}
		return replaceLocationGroups(ctx, tx, id, mutation.GroupIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetLocation(ctx, id)
}

func (s *Store) UpdateLocation(ctx context.Context, id int64, mutation LocationMutation) (*Location, error) {
	if err := s.validateLocationAttachments(ctx, mutation.BackgroundObjectID, mutation.LogoObjectID); err != nil {
		return nil, err
	}
	var oldBackgroundObjectID, oldLogoObjectID *int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT background_object_id, logo_object_id FROM locations WHERE id=$1 FOR UPDATE`, id).
			Scan(&oldBackgroundObjectID, &oldLogoObjectID); err != nil {
			return postgres.GetError(err)
		}
		tag, err := tx.Exec(ctx, `UPDATE locations SET name=$2, description=$3, enabled=$4, notes=$5,
            photo=$6, background_object_id=$7, logo_object_id=$8, updated_at=now() WHERE id=$1`,
			id, mutation.Name, mutation.Description, mutation.Enabled, mutation.Notes, mutation.Photo,
			mutation.BackgroundObjectID, mutation.LogoObjectID)
		if err != nil {
			return postgres.MutationError(err)
		}
		if tag.RowsAffected() == 0 {
			return fault.ErrNotFound
		}
		return replaceLocationGroups(ctx, tx, id, mutation.GroupIDs)
	})
	if err != nil {
		return nil, err
	}
	s.objects.DeleteUnreferenced(ctx,
		replacedObjectIDs(oldBackgroundObjectID, mutation.BackgroundObjectID)...)
	s.objects.DeleteUnreferenced(ctx,
		replacedObjectIDs(oldLogoObjectID, mutation.LogoObjectID)...)
	return s.GetLocation(ctx, id)
}

func replaceLocationGroups(ctx context.Context, tx pgx.Tx, locationID int64, groupIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM location_directory_groups WHERE location_id=$1`, locationID); err != nil {
		return err
	}
	for _, groupID := range groupIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO location_directory_groups (location_id, group_id) VALUES ($1,$2)`, locationID, groupID); err != nil {
			return postgres.MutationError(err)
		}
	}
	return nil
}

func (s *Store) DeleteLocation(ctx context.Context, id int64) error {
	var backgroundObjectID, logoObjectID *int64
	err := s.pool.QueryRow(ctx, `DELETE FROM locations WHERE id=$1 RETURNING background_object_id,logo_object_id`, id).
		Scan(&backgroundObjectID, &logoObjectID)
	if err := postgres.DeleteConflict(err, "location is still referenced"); err != nil {
		return err
	}
	s.objects.DeleteUnreferenced(ctx, pointerIDs(backgroundObjectID, logoObjectID)...)
	return nil
}

func locationFromRow(row locationRow) Location {
	location := Location{ID: row.ID, Name: row.Name, Description: row.Description, Enabled: row.Enabled,
		Notes: row.Notes, Photo: row.Photo, BackgroundObjectID: row.BackgroundObjectID,
		BackgroundURL: backgroundURL(row.BackgroundObjectID), LogoObjectID: row.LogoObjectID,
		LogoURL: logoURL(row.LogoObjectID), Groups: []directory.GroupSummary{}, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	location.BackgroundFile = attachmentFile(row.BackgroundFilename, row.BackgroundType, row.BackgroundSizeBytes, row.BackgroundSHA256)
	location.LogoFile = attachmentFile(row.LogoFilename, row.LogoType, row.LogoSizeBytes, row.LogoSHA256)
	return location
}

type locationGroupQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func attachLocationGroups(ctx context.Context, queryer locationGroupQueryer, locations []Location) error {
	if len(locations) == 0 {
		return nil
	}
	ids := make([]int64, len(locations))
	byID := make(map[int64]*Location, len(locations))
	for i := range locations {
		ids[i] = locations[i].ID
		locations[i].Groups = []directory.GroupSummary{}
		byID[locations[i].ID] = &locations[i]
	}
	rows, err := queryer.Query(ctx, `
SELECT membership.location_id, group_row.id, group_row.source::text,
    group_row.display_name, COALESCE(group_row.mail_nickname, '')
FROM location_directory_groups AS membership
JOIN directory_groups AS group_row ON group_row.id = membership.group_id
WHERE membership.location_id = ANY($1::bigint[])
ORDER BY membership.location_id, lower(group_row.display_name), group_row.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var locationID int64
		var group directory.GroupSummary
		if err := rows.Scan(
			&locationID, &group.ID, &group.Source, &group.DisplayName, &group.MailNickname,
		); err != nil {
			return err
		}
		location := byID[locationID]
		location.Groups = append(location.Groups, group)
	}
	return rows.Err()
}

func (s *Store) SetLocationBackground(ctx context.Context, locationID, objectID int64) error {
	return s.setLocationAttachment(ctx, locationID, objectID, BackgroundObjectPrefix, "background_object_id")
}

func (s *Store) SetLocationLogo(ctx context.Context, locationID, objectID int64) error {
	return s.setLocationAttachment(ctx, locationID, objectID, LogoObjectPrefix, "logo_object_id")
}

func (s *Store) setLocationAttachment(ctx context.Context, locationID, objectID int64, prefix, column string) error {
	if err := s.requireImage(ctx, objectID, prefix); err != nil {
		return err
	}
	var oldObjectID *int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT `+column+` FROM locations WHERE id=$1 FOR UPDATE`, locationID).Scan(&oldObjectID); err != nil {
			return postgres.GetError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE locations SET `+column+`=$2,updated_at=now() WHERE id=$1`, locationID, objectID); err != nil {
			return postgres.MutationError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.objects.DeleteUnreferenced(ctx, replacedObjectIDs(oldObjectID, &objectID)...)
	return nil
}

func (s *Store) validateLocationAttachments(ctx context.Context, backgroundObjectID, logoObjectID *int64) error {
	if backgroundObjectID != nil {
		if err := s.requireImage(ctx, *backgroundObjectID, BackgroundObjectPrefix); err != nil {
			return fmt.Errorf("background: %w", err)
		}
	}
	if logoObjectID != nil {
		if err := s.requireImage(ctx, *logoObjectID, LogoObjectPrefix); err != nil {
			return fmt.Errorf("logo: %w", err)
		}
	}
	return nil
}

func (s *Store) requireImage(ctx context.Context, objectID int64, prefix string) error {
	object, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return err
	}
	if object.Prefix != prefix {
		return fmt.Errorf("%w: object belongs to a different attachment gallery", fault.ErrInvalidInput)
	}
	if !object.Available() {
		return fmt.Errorf("%w: object upload is not complete", fault.ErrInvalidInput)
	}
	detected := mimetype.Lookup(object.ContentType)
	if detected == nil || !detected.Is("image/png") && !detected.Is("image/jpeg") && !detected.Is("image/webp") {
		return fmt.Errorf("%w: attachment must be a PNG, JPEG, or WebP image", fault.ErrInvalidInput)
	}
	return nil
}

type checkinRow struct {
	ID              int64     `db:"id"`
	UserID          int64     `db:"user_id"`
	UserName        string    `db:"user_name"`
	UserEmail       string    `db:"user_email"`
	Department      string    `db:"department"`
	LocationID      int64     `db:"location_id"`
	LocationName    string    `db:"location_name"`
	Direction       string    `db:"direction"`
	Notes           string    `db:"notes"`
	PhotoObjectID   *int64    `db:"photo_object_id"`
	PhotoFilename   *string   `db:"photo_filename"`
	PhotoType       *string   `db:"photo_content_type"`
	PhotoSizeBytes  *int64    `db:"photo_size_bytes"`
	PhotoSHA256     *string   `db:"photo_sha256"`
	StationID       *int64    `db:"station_id"`
	StationName     string    `db:"station_name"`
	CreatedByUserID *int64    `db:"created_by_user_id"`
	CreatedByName   string    `db:"created_by_name"`
	CreatedByEmail  string    `db:"created_by_email"`
	CreatedAt       time.Time `db:"created_at"`
}

const checkinSelectSQL = `SELECT c.id,c.user_id,u.name AS user_name,u.email AS user_email,COALESCE(u.department,'') AS department,
    c.location_id,l.name AS location_name,c.direction::text AS direction,c.notes,c.photo_object_id,
    photo.filename AS photo_filename,photo.content_type AS photo_content_type,
    photo.size_bytes AS photo_size_bytes,photo.sha256 AS photo_sha256,
    c.station_id,COALESCE(station.name,'') AS station_name,
    c.created_by_user_id,COALESCE(creator.name,'') AS created_by_name,
    COALESCE(creator.email,'') AS created_by_email,c.created_at
FROM checkins c
JOIN users u ON u.id=c.user_id
JOIN locations l ON l.id=c.location_id
LEFT JOIN storage_objects photo ON photo.id=c.photo_object_id
LEFT JOIN stations station ON station.id=c.station_id
LEFT JOIN users creator ON creator.id=c.created_by_user_id`

func (s *Store) ListCheckins(ctx context.Context, params CheckinListParams) ([]Checkin, int, error) {
	var where postgres.WhereBuilder
	if params.ListParams.Q != "" {
		where.Addf("(u.name ILIKE '%%'||%s||'%%' OR u.email ILIKE '%%'||%s||'%%' OR l.name ILIKE '%%'||%s||'%%' OR c.notes ILIKE '%%'||%s||'%%')", params.ListParams.Q, params.ListParams.Q, params.ListParams.Q, params.ListParams.Q)
	}
	if params.LocationID > 0 {
		where.Addf("c.location_id=%s", params.LocationID)
	}
	if params.UserID > 0 {
		where.Addf("c.user_id=%s", params.UserID)
	}
	if params.Direction != "" {
		where.Addf("c.direction=%s", params.Direction)
	}
	if params.Department != "" {
		where.Addf("u.department=%s", params.Department)
	}
	if params.CreatedFrom != nil {
		where.Addf("c.created_at >= %s", *params.CreatedFrom)
	}
	if params.CreatedBefore != nil {
		where.Addf("c.created_at < %s", *params.CreatedBefore)
	}
	whereSQL, args := where.Build()
	rows, count, err := postgres.ListWithCount[checkinRow](ctx, s.pool, postgres.ListQuery{SelectSQL: checkinSelectSQL, WhereSQL: whereSQL, Args: args,
		OrderKeys:    map[string]postgres.OrderExpr{"id": {SQL: "c.id"}, "user": {SQL: "lower(u.name)"}, "department": {SQL: "lower(u.department)", NullOrder: postgres.NullsLast}, "location": {SQL: "lower(l.name)"}, "direction": {SQL: "c.direction"}, "created_at": {SQL: "c.created_at"}},
		DefaultOrder: []postgres.OrderExpr{{SQL: "c.created_at", Descending: true}, {SQL: "c.id", Descending: true}}, Params: params.ListParams})
	if err != nil {
		return nil, 0, err
	}
	items := make([]Checkin, len(rows))
	for i, row := range rows {
		items[i] = checkinFromRow(row)
	}
	return items, count, nil
}

func (s *Store) GetCheckin(ctx context.Context, id int64) (*Checkin, error) {
	row, err := postgres.GetOne[checkinRow](ctx, s.pool, checkinSelectSQL+" WHERE c.id=$1", id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	item := checkinFromRow(row)
	return &item, nil
}

func (s *Store) CreateCheckin(ctx context.Context, create CheckinCreate, stationID, createdByUserID, photoObjectID *int64) (*Checkin, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO checkins (user_id,location_id,direction,notes,photo_object_id,station_id,created_by_user_id)
VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, create.UserID, create.LocationID, create.Direction, create.Notes, photoObjectID, stationID, createdByUserID).Scan(&id)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	return s.GetCheckin(ctx, id)
}

func checkinFromRow(row checkinRow) Checkin {
	item := Checkin{
		ID: row.ID,
		Person: PersonSummary{
			ID: row.UserID, Email: row.UserEmail, Name: row.UserName, Department: row.Department,
		},
		Location:      LocationSummary{ID: row.LocationID, Name: row.LocationName},
		Direction:     Direction(row.Direction),
		Notes:         row.Notes,
		PhotoObjectID: row.PhotoObjectID,
		PhotoURL:      photoURL(row.ID, row.PhotoObjectID),
		Station:       stationSummary(row.StationID, row.StationName),
		CreatedBy:     userSummary(row.CreatedByUserID, row.CreatedByName, row.CreatedByEmail),
		CreatedAt:     row.CreatedAt,
	}
	item.PhotoFile = attachmentFile(row.PhotoFilename, row.PhotoType, row.PhotoSizeBytes, row.PhotoSHA256)
	return item
}

func stationSummary(id *int64, name string) *StationSummary {
	if id == nil {
		return nil
	}
	return &StationSummary{ID: *id, Name: name}
}

func userSummary(id *int64, name, email string) *directory.UserSummary {
	if id == nil {
		return nil
	}
	return &directory.UserSummary{ID: *id, Name: name, Email: email}
}

func (s *Store) PersonEligible(ctx context.Context, locationID, userID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, eligiblePersonSQL+" AND u.id=$2", locationID, userID).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return ok, err
}

const eligiblePersonSQL = `SELECT true FROM users u
WHERE u.deleted_at IS NULL
AND (NOT EXISTS (SELECT 1 FROM location_directory_groups WHERE location_id=$1)
 OR EXISTS (SELECT 1 FROM location_directory_groups lg JOIN directory_group_memberships gm ON gm.group_id=lg.group_id WHERE lg.location_id=$1 AND gm.user_id=u.id))`

type personRow struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func (s *Store) listPeople(ctx context.Context, locationID int64) ([]personRow, error) {
	return postgres.GetAll[personRow](ctx, s.pool, `SELECT u.id,u.name,u.email FROM users u WHERE u.deleted_at IS NULL
AND (NOT EXISTS (SELECT 1 FROM location_directory_groups WHERE location_id=$1)
 OR EXISTS (SELECT 1 FROM location_directory_groups lg JOIN directory_group_memberships gm ON gm.group_id=lg.group_id WHERE lg.location_id=$1 AND gm.user_id=u.id)) ORDER BY lower(u.name),u.id`, locationID)
}

func attachmentFile(filename, contentType *string, sizeBytes *int64, sha256 *string) *AttachmentFile {
	if filename == nil || contentType == nil {
		return nil
	}
	file := &AttachmentFile{Filename: *filename, ContentType: *contentType}
	if sizeBytes != nil {
		file.SizeBytes = *sizeBytes
	}
	if sha256 != nil {
		file.SHA256 = *sha256
	}
	return file
}

func replacedObjectIDs(oldID, newID *int64) []int64 {
	if oldID == nil || newID != nil && *oldID == *newID {
		return nil
	}
	return []int64{*oldID}
}

// Kept local to avoid exporting pagination mechanics from this domain package.
func listingNormalize(params listing.Params) listing.Params { return listing.Normalize(params) }
