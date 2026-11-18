-- +goose Up

ALTER TABLE users ADD COLUMN account_enabled BOOLEAN NOT NULL DEFAULT true;

-- A snapshot can move identifiers between objects in a single statement.
-- Deferrable constraints check the resulting directory after those moves.
DROP INDEX users_provider_lower_email_active_idx;
DROP INDEX users_provider_lower_upn_active_idx;

ALTER TABLE users
    ADD CONSTRAINT users_provider_signin_email
        EXCLUDE USING btree (email WITH =)
        WHERE (source <> 'local' AND deleted_at IS NULL AND account_enabled AND email <> '')
        DEFERRABLE INITIALLY IMMEDIATE,
    ADD CONSTRAINT users_provider_current_upn
        EXCLUDE USING btree (user_principal_name WITH =)
        WHERE (source <> 'local' AND deleted_at IS NULL AND user_principal_name <> '')
        DEFERRABLE INITIALLY IMMEDIATE;
