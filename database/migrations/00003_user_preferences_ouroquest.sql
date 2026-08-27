-- +goose Up
ALTER TABLE users_preferences
    ADD COLUMN auto_mudae_oq BOOLEAN NULL AFTER auto_mudae_oc;

-- +goose Down
ALTER TABLE users_preferences
    DROP COLUMN auto_mudae_oq;
