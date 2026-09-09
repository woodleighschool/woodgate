-- +goose Up

-- Move the old tables with their indexes and constraints before reusing names.
CREATE SCHEMA foundation_upgrade;
ALTER TABLE users SET SCHEMA foundation_upgrade;
ALTER TABLE groups SET SCHEMA foundation_upgrade;
ALTER TABLE group_memberships SET SCHEMA foundation_upgrade;
ALTER TABLE assets SET SCHEMA foundation_upgrade;
ALTER TABLE locations SET SCHEMA foundation_upgrade;
ALTER TABLE api_keys SET SCHEMA foundation_upgrade;
ALTER TABLE permissions SET SCHEMA foundation_upgrade;
ALTER TABLE principal_roles SET SCHEMA foundation_upgrade;
ALTER TABLE checkins SET SCHEMA foundation_upgrade;

CREATE TYPE directory_source AS ENUM ('local', 'entra');
CREATE TYPE checkin_direction AS ENUM ('check_in', 'check_out');

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    password_hash TEXT,
    api_key TEXT,
    api_key_created_at TIMESTAMPTZ,
    source directory_source NOT NULL DEFAULT 'local',
    external_id TEXT,
    user_principal_name TEXT,
    mail_nickname TEXT,
    given_name TEXT,
    family_name TEXT,
    department TEXT,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CHECK (
        (source = 'local' AND external_id IS NULL)
        OR (source <> 'local' AND external_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX users_api_key_idx ON users (api_key) WHERE api_key IS NOT NULL;
CREATE INDEX users_department_idx ON users (department) WHERE department IS NOT NULL;
CREATE INDEX users_lower_email_idx ON users (lower(email));
CREATE UNIQUE INDEX users_local_lower_email_active_idx
    ON users (email)
    WHERE email <> '' AND deleted_at IS NULL AND source = 'local';
CREATE UNIQUE INDEX users_provider_lower_email_active_idx
    ON users (email)
    WHERE email <> '' AND deleted_at IS NULL AND source <> 'local';
CREATE UNIQUE INDEX users_provider_lower_upn_active_idx
    ON users (user_principal_name)
    WHERE deleted_at IS NULL AND source <> 'local' AND user_principal_name IS NOT NULL AND user_principal_name <> '';

-- Owned by alexedwards/scs/pgxstore.
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BYTEA NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

CREATE TABLE directory_groups (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    source directory_source NOT NULL,
    external_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    mail_nickname TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CHECK (source <> 'local')
);

CREATE TABLE directory_group_memberships (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES directory_groups (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX directory_group_memberships_group_idx ON directory_group_memberships (group_id);

CREATE TABLE locations (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT true,
    notes BOOLEAN NOT NULL DEFAULT false,
    photo BOOLEAN NOT NULL DEFAULT false,
    background_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    background_app_id UUID UNIQUE,
    logo_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    logo_app_id UUID UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (btrim(name) <> ''),
    CHECK ((background_object_id IS NULL) = (background_app_id IS NULL)),
    CHECK ((logo_object_id IS NULL) = (logo_app_id IS NULL))
);

CREATE INDEX locations_background_object_idx ON locations (background_object_id);
CREATE INDEX locations_logo_object_idx ON locations (logo_object_id);

CREATE TABLE location_directory_groups (
    location_id BIGINT NOT NULL REFERENCES locations (id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES directory_groups (id) ON DELETE CASCADE,
    PRIMARY KEY (location_id, group_id)
);

CREATE INDEX location_directory_groups_group_idx ON location_directory_groups (group_id);

CREATE TABLE app_keys (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    all_locations BOOLEAN NOT NULL DEFAULT false,
    read_users BOOLEAN NOT NULL DEFAULT true,
    read_locations BOOLEAN NOT NULL DEFAULT true,
    read_branding BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE app_key_locations (
    app_key_id BIGINT NOT NULL REFERENCES app_keys (id) ON DELETE CASCADE,
    location_id BIGINT NOT NULL REFERENCES locations (id) ON DELETE CASCADE,
    PRIMARY KEY (app_key_id, location_id)
);
CREATE INDEX app_key_locations_location_idx ON app_key_locations (location_id);

CREATE TABLE checkins (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users (id),
    location_id BIGINT NOT NULL REFERENCES locations (id),
    direction checkin_direction NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    photo_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    app_key_id BIGINT REFERENCES app_keys (id) ON DELETE SET NULL,
    actor_kind TEXT NOT NULL CHECK (actor_kind IN ('user', 'api_key')),
    actor_app_id UUID NOT NULL,
    created_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX checkins_location_created_idx ON checkins (location_id, created_at DESC, id DESC);
CREATE INDEX checkins_user_created_idx ON checkins (user_id, created_at DESC, id DESC);
CREATE INDEX checkins_photo_object_idx ON checkins (photo_object_id);

CREATE TABLE authz_roles (
    id BIGSERIAL PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    builtin BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (btrim(key) <> ''),
    CHECK (btrim(name) <> '')
);

CREATE TABLE authz_role_permissions (
    role_id BIGINT NOT NULL REFERENCES authz_roles (id) ON DELETE CASCADE,
    resource TEXT NOT NULL,
    access SMALLINT NOT NULL CHECK (access IN (1, 2)),
    PRIMARY KEY (role_id, resource)
);

CREATE TABLE authz_user_roles (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES authz_roles (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE authz_group_roles (
    group_id BIGINT NOT NULL REFERENCES directory_groups (id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES authz_roles (id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, role_id)
);

INSERT INTO authz_roles (key, name, description, builtin)
VALUES ('admin', 'Admin', 'Full access to every resource.', true);

INSERT INTO authz_role_permissions (role_id, resource, access)
SELECT id, resource, 2
FROM authz_roles
CROSS JOIN unnest(ARRAY[
    'users',
    'groups',
    'directory',
    'locations',
    'checkins',
    'app_keys',
    'authz.roles'
]) AS resource
WHERE key = 'admin';

INSERT INTO users (
    app_id, email, name, source, external_id, user_principal_name,
    department, created_at, updated_at
)
-- The old service writes users only through Entra; missing users become "local".
-- Keep their provider identity so a returning user reuses the same history and roles.
SELECT id, upn, display_name, 'entra', id::text,
    NULLIF(upn, ''), department, created_at, updated_at
FROM foundation_upgrade.users;

-- Groups in the v1 service are written only by Entra reconciliation.
INSERT INTO directory_groups (app_id, source, external_id, display_name, description, created_at, updated_at)
SELECT id, 'entra', id::text, name, description, created_at, updated_at
FROM foundation_upgrade.groups;

INSERT INTO directory_group_memberships (user_id, group_id)
SELECT u.id, g.id
FROM foundation_upgrade.group_memberships old
JOIN users u ON u.app_id = old.user_id
JOIN directory_groups g ON g.app_id = old.group_id;

INSERT INTO locations (app_id, name, description, enabled, notes, photo, created_at, updated_at)
SELECT id, name, description, enabled, notes, photo, created_at, updated_at
FROM foundation_upgrade.locations;

INSERT INTO location_directory_groups (location_id, group_id)
SELECT DISTINCT l.id, g.id
FROM foundation_upgrade.locations old
CROSS JOIN LATERAL unnest(old.group_ids) AS member(group_id)
JOIN locations l ON l.app_id = old.id
JOIN directory_groups g ON g.app_id = member.group_id;

INSERT INTO app_keys (app_id, name, key_prefix, key_hash, all_locations, read_users, read_locations, read_branding, expires_at, last_used_at, created_at, updated_at)
SELECT old.id, old.name, old.key_prefix, old.key_hash,
    EXISTS (
        SELECT 1 FROM foundation_upgrade.principal_roles r
        WHERE r.principal_kind = 'api_key' AND r.principal_id = old.id AND r.role = 'admin'
    ) OR EXISTS (
        SELECT 1 FROM foundation_upgrade.permissions p
        WHERE p.subject_kind = 'api_key' AND p.subject_id = old.id
          AND p.resource = 'checkins' AND p.action = 'create'
          AND p.location_id IS NULL
    ),
    EXISTS (
        SELECT 1 FROM foundation_upgrade.principal_roles r
        WHERE r.principal_kind = 'api_key' AND r.principal_id = old.id AND r.role = 'admin'
    ) OR EXISTS (
        SELECT 1 FROM foundation_upgrade.permissions p
        WHERE p.subject_kind = 'api_key' AND p.subject_id = old.id
          AND p.resource = 'users' AND p.action = 'read'
    ),
    EXISTS (
        SELECT 1 FROM foundation_upgrade.principal_roles r
        WHERE r.principal_kind = 'api_key' AND r.principal_id = old.id AND r.role = 'admin'
    ) OR EXISTS (
        SELECT 1 FROM foundation_upgrade.permissions p
        WHERE p.subject_kind = 'api_key' AND p.subject_id = old.id
          AND p.resource = 'locations' AND p.action = 'read'
    ),
    EXISTS (
        SELECT 1 FROM foundation_upgrade.principal_roles r
        WHERE r.principal_kind = 'api_key' AND r.principal_id = old.id AND r.role = 'admin'
    ) OR EXISTS (
        SELECT 1 FROM foundation_upgrade.permissions p
        WHERE p.subject_kind = 'api_key' AND p.subject_id = old.id
          AND p.resource = 'assets' AND p.action = 'read'
          AND (p.asset_type IS NULL OR p.asset_type = 'asset')
    ),
    old.expires_at, old.last_used_at, old.created_at, old.created_at
FROM foundation_upgrade.api_keys old;

INSERT INTO app_key_locations (app_key_id, location_id)
SELECT DISTINCT k.id, l.id
FROM foundation_upgrade.permissions p
JOIN app_keys k ON k.app_id = p.subject_id
JOIN locations l ON l.app_id = p.location_id
WHERE p.subject_kind = 'api_key' AND p.resource = 'checkins'
  AND p.action = 'create';

INSERT INTO checkins (
    app_id, user_id, location_id, direction, notes,
    actor_kind, actor_app_id, created_by_user_id, app_key_id, created_at
)
SELECT old.id, u.id, l.id, old.direction::checkin_direction, old.notes,
    old.created_by_kind, old.created_by_id, author.id, k.id, old.created_at
FROM foundation_upgrade.checkins old
JOIN users u ON u.app_id = old.user_id
JOIN locations l ON l.app_id = old.location_id
LEFT JOIN users author ON old.created_by_kind = 'user' AND author.app_id = old.created_by_id
LEFT JOIN app_keys k ON old.created_by_kind = 'api_key' AND k.app_id = old.created_by_id;

INSERT INTO authz_user_roles (user_id, role_id)
SELECT u.id, r.id
FROM foundation_upgrade.principal_roles old
JOIN users u ON u.app_id = old.principal_id
CROSS JOIN authz_roles r
WHERE old.principal_kind = 'user' AND old.role = 'admin' AND r.key = 'admin';

-- Abort rather than commit an incomplete history or identity translation.
-- +goose StatementBegin
DO $$
BEGIN
    IF (SELECT count(*) FROM users) <> (SELECT count(*) FROM foundation_upgrade.users)
       OR (SELECT count(*) FROM directory_groups) <> (SELECT count(*) FROM foundation_upgrade.groups)
       OR (SELECT count(*) FROM directory_group_memberships) <> (SELECT count(*) FROM foundation_upgrade.group_memberships)
       OR (SELECT count(*) FROM locations) <> (SELECT count(*) FROM foundation_upgrade.locations)
       OR (SELECT count(*) FROM app_keys) <> (SELECT count(*) FROM foundation_upgrade.api_keys)
       OR (SELECT count(*) FROM checkins) <> (SELECT count(*) FROM foundation_upgrade.checkins) THEN
        RAISE EXCEPTION 'Shared foundations migration did not preserve every identity and check-in';
    END IF;
END $$;
-- +goose StatementEnd

-- Attachment references are intentionally discarded; Bloby owns all new objects.
DROP SCHEMA foundation_upgrade CASCADE;
