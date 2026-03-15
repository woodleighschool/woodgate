-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  upn TEXT NOT NULL DEFAULT '',
  display_name TEXT NOT NULL DEFAULT '',
  department TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT users_source_check CHECK (source IN ('local', 'entra'))
);

CREATE INDEX IF NOT EXISTS users_source_idx ON users (source);
CREATE INDEX IF NOT EXISTS users_upn_idx ON users (upn);
CREATE UNIQUE INDEX IF NOT EXISTS users_upn_unique_idx ON users (upn) WHERE upn <> '';

CREATE TABLE IF NOT EXISTS groups (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS groups_name_idx ON groups (name);

CREATE TABLE IF NOT EXISTS group_memberships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  group_id UUID NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT group_memberships_unique UNIQUE (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS group_memberships_group_idx ON group_memberships (group_id);
CREATE INDEX IF NOT EXISTS group_memberships_user_idx ON group_memberships (user_id);

CREATE TABLE IF NOT EXISTS assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NULL,
  type TEXT NOT NULL,
  content_type TEXT NOT NULL,
  file_extension TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT assets_name_check CHECK (name IS NULL OR btrim(name) <> ''),
  CONSTRAINT assets_type_check CHECK (type IN ('asset', 'photo')),
  CONSTRAINT assets_content_type_check CHECK (content_type IN ('image/png', 'image/jpeg')),
  CONSTRAINT assets_file_extension_check CHECK (file_extension IN ('.png', '.jpg'))
);

CREATE TABLE IF NOT EXISTS locations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  notes BOOLEAN NOT NULL DEFAULT FALSE,
  photo BOOLEAN NOT NULL DEFAULT FALSE,
  background_asset_id UUID NULL REFERENCES assets (id) ON DELETE SET NULL,
  logo_asset_id UUID NULL REFERENCES assets (id) ON DELETE SET NULL,
  group_ids UUID[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT locations_name_check CHECK (btrim(name) <> '')
);

CREATE INDEX IF NOT EXISTS locations_enabled_idx ON locations (enabled);
CREATE INDEX IF NOT EXISTS locations_name_idx ON locations (name);

CREATE TABLE IF NOT EXISTS api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  key_prefix TEXT NOT NULL,
  key_hash TEXT NOT NULL UNIQUE,
  last_used_at TIMESTAMPTZ NULL,
  expires_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT api_keys_name_check CHECK (btrim(name) <> ''),
  CONSTRAINT api_keys_prefix_check CHECK (btrim(key_prefix) <> ''),
  CONSTRAINT api_keys_hash_check CHECK (btrim(key_hash) <> '')
);

CREATE INDEX IF NOT EXISTS api_keys_name_idx ON api_keys (name);
CREATE INDEX IF NOT EXISTS api_keys_expires_at_idx ON api_keys (expires_at);

CREATE TABLE IF NOT EXISTS permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subject_kind TEXT NOT NULL,
  subject_id UUID NOT NULL,
  resource TEXT NOT NULL,
  action TEXT NOT NULL,
  location_id UUID NULL REFERENCES locations (id) ON DELETE CASCADE,
  asset_type TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT permissions_subject_kind_check CHECK (subject_kind IN ('user', 'api_key')),
  CONSTRAINT permissions_resource_check CHECK (
    resource IN ('users', 'groups', 'locations', 'checkins', 'assets', 'api_keys')
  ),
  CONSTRAINT permissions_action_check CHECK (action IN ('read', 'create', 'write', 'delete')),
  CONSTRAINT permissions_asset_type_check CHECK (asset_type IS NULL OR asset_type IN ('asset', 'photo')),
  CONSTRAINT permissions_scope_check CHECK (
    (resource = 'checkins' AND location_id IS NOT NULL AND asset_type IS NULL)
    OR (resource = 'assets' AND location_id IS NULL AND asset_type IS NOT NULL)
    OR (resource NOT IN ('checkins', 'assets') AND location_id IS NULL AND asset_type IS NULL)
  )
);

CREATE TABLE IF NOT EXISTS principal_roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  principal_kind TEXT NOT NULL,
  principal_id UUID NOT NULL,
  role TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT principal_roles_principal_kind_check CHECK (principal_kind IN ('user', 'api_key')),
  CONSTRAINT principal_roles_role_check CHECK (role IN ('admin')),
  CONSTRAINT principal_roles_unique_idx UNIQUE (principal_kind, principal_id, role)
);

CREATE UNIQUE INDEX IF NOT EXISTS permissions_subject_resource_unique_idx
  ON permissions (subject_kind, subject_id, resource)
  WHERE location_id IS NULL AND asset_type IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS permissions_subject_resource_location_unique_idx
  ON permissions (subject_kind, subject_id, resource, location_id)
  WHERE location_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS permissions_subject_resource_asset_type_unique_idx
  ON permissions (subject_kind, subject_id, resource, asset_type)
  WHERE asset_type IS NOT NULL;
CREATE INDEX IF NOT EXISTS permissions_subject_idx ON permissions (subject_kind, subject_id);
CREATE INDEX IF NOT EXISTS permissions_location_idx ON permissions (location_id);
CREATE INDEX IF NOT EXISTS permissions_asset_type_idx ON permissions (asset_type);
CREATE INDEX IF NOT EXISTS principal_roles_principal_idx ON principal_roles (principal_kind, principal_id);

CREATE TABLE IF NOT EXISTS checkins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users (id),
  location_id UUID NOT NULL REFERENCES locations (id),
  direction TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  asset_id UUID NULL REFERENCES assets (id) ON DELETE SET NULL,
  created_by_kind TEXT NOT NULL,
  created_by_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT checkins_direction_check CHECK (direction IN ('check_in', 'check_out')),
  CONSTRAINT checkins_created_by_kind_check CHECK (created_by_kind IN ('user', 'api_key'))
);

CREATE INDEX IF NOT EXISTS checkins_location_created_idx ON checkins (location_id, created_at DESC);
CREATE INDEX IF NOT EXISTS checkins_user_created_idx ON checkins (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS checkins_direction_created_idx ON checkins (direction, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS checkins_asset_unique_idx
  ON checkins (asset_id)
  WHERE asset_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS checkins;
DROP TABLE IF EXISTS principal_roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS group_memberships;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS pgcrypto;
