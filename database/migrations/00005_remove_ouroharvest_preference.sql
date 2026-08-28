-- +goose Up
ALTER TABLE users_preferences
    DROP COLUMN auto_mudae_oh;

-- +goose Down
ALTER TABLE users_preferences
    ADD COLUMN auto_mudae_oh BOOLEAN NULL AFTER auto_mudae_oq;
