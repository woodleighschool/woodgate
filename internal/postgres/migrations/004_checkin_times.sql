-- +goose Up

ALTER TABLE checkins ADD COLUMN occurred_at TIMESTAMPTZ;
ALTER TABLE checkins ADD COLUMN updated_at TIMESTAMPTZ;
UPDATE checkins SET occurred_at = created_at, updated_at = created_at;
ALTER TABLE checkins ALTER COLUMN occurred_at SET NOT NULL;
ALTER TABLE checkins ALTER COLUMN occurred_at SET DEFAULT now();
ALTER TABLE checkins ALTER COLUMN updated_at SET NOT NULL;
ALTER TABLE checkins ALTER COLUMN updated_at SET DEFAULT now();

DROP INDEX checkins_location_created_idx;
DROP INDEX checkins_user_created_idx;
CREATE INDEX checkins_location_occurred_idx ON checkins (location_id, occurred_at DESC, id DESC);
CREATE INDEX checkins_user_occurred_idx ON checkins (user_id, occurred_at DESC, id DESC);
