package directory

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/postgres"
)

// Store persists directory users, groups, memberships, and source snapshots.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a directory store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type userRow struct {
	ID                int64      `db:"id"`
	Email             string     `db:"email"`
	Name              string     `db:"name"`
	PasswordHash      *string    `db:"password_hash"`
	APIKey            *string    `db:"api_key"`
	APIKeyCreatedAt   *time.Time `db:"api_key_created_at"`
	Source            string     `db:"source"`
	ExternalID        *string    `db:"external_id"`
	UserPrincipalName *string    `db:"user_principal_name"`
	MailNickname      *string    `db:"mail_nickname"`
	GivenName         *string    `db:"given_name"`
	FamilyName        *string    `db:"family_name"`
	Department        *string    `db:"department"`
	DeletedAt         *time.Time `db:"deleted_at"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

var userColumnExprs = []string{
	"id",
	"email",
	"name",
	"password_hash",
	"api_key",
	"api_key_created_at",
	"source::text AS source",
	"external_id",
	"user_principal_name",
	"mail_nickname",
	"given_name",
	"family_name",
	"department",
	"deleted_at",
	"created_at",
	"updated_at",
}

func userColumnsSQL(alias string) string {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	columns := make([]string, len(userColumnExprs))
	for i, column := range userColumnExprs {
		columns[i] = prefix + column
	}
	return strings.Join(columns, ", ")
}

func userSelectSQL() string {
	return `
SELECT
    ` + userColumnsSQL("") + `
FROM users`
}

func userFromRow(r userRow) User {
	return User{
		ID:                r.ID,
		Email:             r.Email,
		Name:              r.Name,
		PasswordHash:      derefString(r.PasswordHash),
		Source:            Source(r.Source),
		ExternalID:        derefString(r.ExternalID),
		UserPrincipalName: derefString(r.UserPrincipalName),
		MailNickname:      derefString(r.MailNickname),
		GivenName:         derefString(r.GivenName),
		FamilyName:        derefString(r.FamilyName),
		Department:        derefString(r.Department),
		DeletedAt:         r.DeletedAt,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func accountFromRow(r userRow) Account {
	account := Account{
		User:            userFromRow(r),
		APIKeyCreatedAt: r.APIKeyCreatedAt,
	}
	if r.APIKey != nil {
		account.APIKey = *r.APIKey
	}
	return account
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type userCreateRecord struct {
	Email        string
	Name         string
	PasswordHash string
	RoleIDs      []int64
}

func (s *Store) createUser(
	ctx context.Context,
	params userCreateRecord,
) (*User, error) {
	var user User
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		row, err := postgres.GetOne[userRow](ctx, tx, `
INSERT INTO users (email, name, password_hash, source)
VALUES ($1, $2, $3, 'local')
RETURNING `+userColumnsSQL(""), params.Email, params.Name, params.PasswordHash)
		if err != nil {
			return postgres.MutationError(err)
		}
		users := []User{userFromRow(row)}
		if err := replaceUserRoles(ctx, tx, users[0].ID, params.RoleIDs); err != nil {
			return err
		}
		if err := attachUserRoles(ctx, tx, users); err != nil {
			return err
		}
		user = users[0]
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row, err := postgres.GetOne[userRow](ctx, s.pool, userSelectSQL()+`
WHERE id = $1
  AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	out := userFromRow(row)
	users := []User{out}
	if err := attachUserRoles(ctx, s.pool, users); err != nil {
		return nil, err
	}
	out = users[0]
	return &out, nil
}

// GetAccountByID returns the signed-in user's self-view, including API key fields.
func (s *Store) GetAccountByID(ctx context.Context, id int64) (*Account, error) {
	row, err := postgres.GetOne[userRow](ctx, s.pool, userSelectSQL()+`
WHERE id = $1
  AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	out := accountFromRow(row)
	users := []User{out.User}
	if err := attachUserRoles(ctx, s.pool, users); err != nil {
		return nil, err
	}
	out.User = users[0]
	return &out, nil
}

func (s *Store) ListUsers(ctx context.Context, params UserListParams) ([]User, int, error) {
	where, args := userWhere(params)
	rows, count, err := postgres.ListWithCount[userRow](ctx, s.pool, userListQuery(params, where, args))
	if err != nil {
		return nil, 0, err
	}
	out := make([]User, len(rows))
	for i, row := range rows {
		out[i] = userFromRow(row)
	}
	if err := attachUserRoles(ctx, s.pool, out); err != nil {
		return nil, 0, err
	}
	return out, count, nil
}

func (s *Store) ListRoleSummaries(ctx context.Context) ([]RoleSummary, error) {
	return postgres.GetAll[RoleSummary](ctx, s.pool, `
SELECT id, key, name
FROM authz_roles
ORDER BY builtin DESC, lower(name), id`)
}

type userUpdateRecord struct {
	Name         string
	PasswordHash *string
	RoleIDs      []int64
}

func (s *Store) updateUser(ctx context.Context, id int64, params userUpdateRecord) (*User, error) {
	var user User
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		row, err := postgres.GetOne[userRow](ctx, tx, `
UPDATE users
SET
    name = CASE WHEN source = 'local' THEN $1 ELSE name END,
    password_hash = CASE
        WHEN source = 'local' THEN COALESCE($2, password_hash)
        ELSE password_hash
    END,
    updated_at = now()
WHERE id = $3
  AND deleted_at IS NULL
RETURNING `+userColumnsSQL(""),
			params.Name, params.PasswordHash, id)
		if err != nil {
			return postgres.MutationError(err)
		}
		users := []User{userFromRow(row)}
		if err := replaceUserRoles(ctx, tx, id, params.RoleIDs); err != nil {
			return err
		}
		if err := attachUserRoles(ctx, tx, users); err != nil {
			return err
		}
		user = users[0]
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) setLocalUserPasswordByEmail(
	ctx context.Context,
	email string,
	passwordHash string,
) (*User, error) {
	qrows, err := s.pool.Query(ctx, `
UPDATE users
SET
    password_hash = $1,
    updated_at = now()
WHERE email = $2
  AND source = 'local'
  AND deleted_at IS NULL
RETURNING `+userColumnsSQL(""), passwordHash, email)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	row, err := pgx.CollectExactlyOneRow(qrows, pgx.RowToStructByName[userRow])
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	user := userFromRow(row)
	users := []User{user}
	if err := attachUserRoles(ctx, s.pool, users); err != nil {
		return nil, err
	}
	user = users[0]
	return &user, nil
}

func (s *Store) setUserRolesByEmail(
	ctx context.Context,
	email string,
	roleIDs []int64,
) (*User, error) {
	var userID int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
SELECT id
FROM users
WHERE lower(email) = lower($1)
  AND deleted_at IS NULL
ORDER BY CASE WHEN source = 'local' THEN 1 ELSE 0 END, id
LIMIT 1
FOR UPDATE`, email).Scan(&userID); err != nil {
			return postgres.GetError(err)
		}
		if err := replaceUserRoles(ctx, tx, userID, roleIDs); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE users SET updated_at = now() WHERE id = $1`, userID)
		return postgres.MutationError(err)
	})
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, userID)
}

func (s *Store) updateAccount(ctx context.Context, id int64, params accountUpdateRecord) (*Account, error) {
	qrows, err := s.pool.Query(ctx, `
UPDATE users
SET
    name = CASE WHEN source = 'local' THEN $1 ELSE name END,
    password_hash = CASE
        WHEN source = 'local' THEN COALESCE($2, password_hash)
        ELSE password_hash
    END,
    updated_at = now()
WHERE id = $3
RETURNING `+userColumnsSQL(""),
		params.Name, params.PasswordHash, id)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	row, err := pgx.CollectExactlyOneRow(qrows, pgx.RowToStructByName[userRow])
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	out := accountFromRow(row)
	users := []User{out.User}
	if err := attachUserRoles(ctx, s.pool, users); err != nil {
		return nil, err
	}
	out.User = users[0]
	return &out, nil
}

func (s *Store) deleteUser(
	ctx context.Context,
	id int64,
) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var source Source
		if err := tx.QueryRow(ctx, `
SELECT source::text
FROM users
WHERE id = $1
  AND deleted_at IS NULL
FOR UPDATE`, id).Scan(&source); err != nil {
			return postgres.GetError(err)
		}

		var deletedID int64
		if source == SourceLocal {
			if err := tx.QueryRow(ctx, `
DELETE FROM users
WHERE id = $1
RETURNING id`, id).Scan(&deletedID); err != nil {
				return postgres.MutationError(err)
			}
		} else {
			if err := tx.QueryRow(ctx, `
UPDATE users
SET
    deleted_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING id`, id).Scan(&deletedID); err != nil {
				return postgres.MutationError(err)
			}
		}
		return nil
	})
}

func userWhere(params UserListParams) (string, []any) {
	var where postgres.WhereBuilder
	where.Add("u.deleted_at IS NULL")
	if params.GroupID > 0 {
		where.Addf("gm.group_id = %s", params.GroupID)
	}
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add(`(
			u.email ILIKE ` + search + `
			OR u.user_principal_name ILIKE ` + search + `
			OR u.mail_nickname ILIKE ` + search + `
			OR u.name ILIKE ` + search + `
			OR u.given_name ILIKE ` + search + `
			OR u.family_name ILIKE ` + search + `
			OR u.department ILIKE ` + search + `
		)`)
	}
	if len(params.Values) > 0 {
		where.Addf("u.id::text = ANY(%s::text[])", listing.NormalizeValues(params.Values))
	}
	if len(params.RoleValues) > 0 {
		where.Add(roleFilterSQL(&where, params.RoleValues))
	}
	switch params.Source {
	case string(SourceLocal):
		where.Add("u.source = 'local'")
	case string(SourceEntra):
		where.Add("u.source = 'entra'")
	}
	return where.Build()
}

func userListQuery(params UserListParams, where string, args []any) postgres.ListQuery {
	return postgres.ListQuery{
		SelectSQL: userListSelectSQL(params),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			"name":       {SQL: "lower(u.name)"},
			"email":      {SQL: "lower(u.email)"},
			"department": {SQL: "lower(u.department)", NullOrder: postgres.NullsLast},
			"created_at": {SQL: "u.created_at"},
			"updated_at": {SQL: "u.updated_at"},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: "lower(u.name)"}, {SQL: "lower(u.email)"}, {SQL: "u.id"}},
		Params:       params.ListParams,
	}
}

func effectiveRoleExistsSQL(userAlias string) string {
	if userAlias != "" {
		userAlias += "."
	}
	return `EXISTS (
    SELECT 1 FROM authz_user_roles AS direct_role WHERE direct_role.user_id = ` + userAlias + `id
    UNION ALL
    SELECT 1
    FROM directory_group_memberships AS membership
    JOIN authz_group_roles AS group_role ON group_role.group_id = membership.group_id
    WHERE membership.user_id = ` + userAlias + `id
)`
}

func roleFilterSQL(where *postgres.WhereBuilder, values []string) string {
	roleIDs := make([]int64, 0, len(values))
	includeNone := false
	for _, value := range values {
		if value == "none" {
			includeNone = true
			continue
		}
		id, _ := strconv.ParseInt(value, 10, 64)
		roleIDs = append(roleIDs, id)
	}
	conditions := make([]string, 0, 2)
	if len(roleIDs) > 0 {
		roleArg := where.Arg(roleIDs)
		conditions = append(conditions, `EXISTS (
            SELECT 1 FROM authz_user_roles AS direct_role
            WHERE direct_role.user_id = u.id AND direct_role.role_id = ANY(`+roleArg+`::bigint[])
            UNION ALL
            SELECT 1
            FROM directory_group_memberships AS membership
            JOIN authz_group_roles AS group_role ON group_role.group_id = membership.group_id
            WHERE membership.user_id = u.id AND group_role.role_id = ANY(`+roleArg+`::bigint[])
        )`)
	}
	if includeNone {
		conditions = append(conditions, "NOT "+effectiveRoleExistsSQL("u"))
	}
	return "(" + strings.Join(conditions, " OR ") + ")"
}

func userListSelectSQL(params UserListParams) string {
	selectSQL := `SELECT ` + userColumnsSQL("u") + `
FROM users u`
	if params.GroupID <= 0 {
		return selectSQL
	}
	return selectSQL + `
JOIN directory_group_memberships gm ON gm.user_id = u.id`
}
