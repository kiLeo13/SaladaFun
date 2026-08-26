-- +goose Up
CREATE TABLE users_preferences (
    user_id BIGINT UNSIGNED NOT NULL,
    auto_mudae_oc BOOLEAN NULL,
    created_at BIGINT UNSIGNED NOT NULL,
    updated_at BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (user_id)
);

-- +goose Down
DROP TABLE users_preferences;
