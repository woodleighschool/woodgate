-- +goose Up

CREATE TABLE stations (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
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

ALTER TABLE checkins ADD COLUMN station_id BIGINT REFERENCES stations (id) ON DELETE SET NULL;
ALTER TABLE checkins DROP CONSTRAINT checkins_actor_kind_check;
ALTER TABLE checkins ADD CONSTRAINT checkins_actor_kind_check
    CHECK (actor_kind IN ('user', 'api_key', 'station'));

INSERT INTO authz_role_permissions (role_id, resource, access)
SELECT role_id, 'stations', access FROM authz_role_permissions WHERE resource = 'app_keys'
ON CONFLICT (role_id, resource) DO UPDATE
    SET access = greatest(authz_role_permissions.access, EXCLUDED.access);
DELETE FROM authz_role_permissions WHERE resource = 'app_keys';
