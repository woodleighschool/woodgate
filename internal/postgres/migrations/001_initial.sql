-- +goose Up

CREATE TYPE directory_source AS ENUM ('local', 'entra');
CREATE TYPE checkin_direction AS ENUM ('check_in', 'check_out');

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
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
    secret_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (btrim(name) <> ''),
    CHECK (secret_hash ~ '^[0-9a-f]{64}$')
);

CREATE INDEX stations_location_idx ON stations (location_id);

CREATE TABLE station_sessions (
    station_id BIGINT PRIMARY KEY REFERENCES stations (id) ON DELETE CASCADE,
    connection_id TEXT NOT NULL,
    protocol_version INTEGER NOT NULL CHECK (protocol_version >= 0),
    version TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE station_rejections (
    station_id BIGINT PRIMARY KEY REFERENCES stations (id) ON DELETE CASCADE,
    protocol_version INTEGER CHECK (protocol_version >= 0),
    version TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

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
    'stations',
    'authz.roles'
]) AS resource
WHERE key = 'admin';
