-- +goose Up

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE directory_source AS ENUM ('local', 'entra');
CREATE TYPE checkin_direction AS ENUM ('check_in', 'check_out');

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    password_hash TEXT,
    access_enabled BOOLEAN NOT NULL DEFAULT false,
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

CREATE INDEX users_department_idx ON users (department) WHERE department IS NOT NULL;
CREATE INDEX users_lower_email_idx ON users (lower(email));
CREATE UNIQUE INDEX users_local_lower_email_active_idx
    ON users (lower(email))
    WHERE deleted_at IS NULL AND source = 'local';
CREATE UNIQUE INDEX users_provider_lower_email_active_idx
    ON users (lower(email))
    WHERE deleted_at IS NULL AND source <> 'local';
CREATE UNIQUE INDEX users_provider_lower_upn_active_idx
    ON users (lower(user_principal_name))
    WHERE deleted_at IS NULL AND source <> 'local' AND user_principal_name IS NOT NULL;

-- Owned by alexedwards/scs/pgxstore.
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BYTEA NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

CREATE TABLE directory_groups (
    id BIGSERIAL PRIMARY KEY,
    source directory_source NOT NULL,
    external_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
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

CREATE TABLE storage_objects (
    id BIGSERIAL PRIMARY KEY,
    prefix TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT CHECK (size_bytes IS NULL OR size_bytes >= 0),
    sha256 TEXT CHECK (sha256 IS NULL OR sha256 ~ '^[0-9a-f]{64}$'),
    multipart_upload_id TEXT CHECK (multipart_upload_id IS NULL OR btrim(multipart_upload_id) <> ''),
    available_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT storage_objects_state_check CHECK (
        (
            available_at IS NULL
            AND content_type = ''
            AND size_bytes IS NULL
            AND sha256 IS NULL
        )
        OR (
            available_at IS NOT NULL
            AND content_type <> ''
            AND size_bytes IS NOT NULL
            AND sha256 IS NOT NULL
            AND multipart_upload_id IS NULL
        )
    ),
    CONSTRAINT storage_objects_expiry_check CHECK (expired_at IS NULL OR available_at IS NULL)
);

CREATE INDEX storage_objects_prefix_idx ON storage_objects (prefix);
CREATE INDEX storage_objects_pending_expiry_idx
    ON storage_objects (updated_at, id)
    WHERE available_at IS NULL;

CREATE TABLE locations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT true,
    notes BOOLEAN NOT NULL DEFAULT false,
    photo BOOLEAN NOT NULL DEFAULT false,
    background_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    logo_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (btrim(name) <> '')
);

CREATE INDEX locations_background_object_idx ON locations (background_object_id);
CREATE INDEX locations_logo_object_idx ON locations (logo_object_id);

CREATE TABLE location_directory_groups (
    location_id BIGINT NOT NULL REFERENCES locations (id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES directory_groups (id) ON DELETE CASCADE,
    PRIMARY KEY (location_id, group_id)
);

CREATE INDEX location_directory_groups_group_idx ON location_directory_groups (group_id);

CREATE TABLE stations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    location_id BIGINT NOT NULL REFERENCES locations (id) ON DELETE RESTRICT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    secret_prefix TEXT NOT NULL,
    secret_hash TEXT NOT NULL UNIQUE,
    last_seen_at TIMESTAMPTZ,
    app_build TEXT NOT NULL DEFAULT '',
    protocol_version INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (btrim(name) <> ''),
    CHECK (btrim(secret_prefix) <> ''),
    CHECK (secret_hash ~ '^[0-9a-f]{64}$'),
    CHECK (protocol_version IS NULL OR protocol_version >= 0)
);

CREATE INDEX stations_location_idx ON stations (location_id);

CREATE TABLE checkins (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id),
    location_id BIGINT NOT NULL REFERENCES locations (id),
    direction checkin_direction NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    photo_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    station_id BIGINT REFERENCES stations (id) ON DELETE SET NULL,
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
VALUES ('owner', 'Owner', 'Full access to every resource.', true);

INSERT INTO authz_role_permissions (role_id, resource, access)
SELECT id, resource, 2
FROM authz_roles
CROSS JOIN unnest(ARRAY[
    'users',
    'groups',
    'directory',
    'locations',
    'checkins',
    'stations',
    'authz.roles',
    'authz.assignments'
]) AS resource
WHERE key = 'owner';

-- Temporary protocol-v0 UUID bridge. The prepared v0-removal change drops
-- this table with the legacy routes after every enabled station reports v1.
CREATE TABLE station_v0_mappings (
    kind TEXT NOT NULL CHECK (kind IN ('user', 'location', 'asset', 'checkin', 'api_key')),
    legacy_id UUID NOT NULL,
    object_id BIGINT NOT NULL,
    PRIMARY KEY (kind, legacy_id),
    UNIQUE (kind, object_id)
);
