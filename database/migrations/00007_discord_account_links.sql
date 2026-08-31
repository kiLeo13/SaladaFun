-- +goose Up
CREATE TABLE discord_account_links (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    parent_id BIGINT UNSIGNED NULL,
    created_at BIGINT UNSIGNED NOT NULL,
    updated_at BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT uq_discord_account_links_user_id UNIQUE (user_id),
    KEY idx_discord_account_links_parent_id (parent_id),
    CONSTRAINT fk_discord_account_links_parent
        FOREIGN KEY (parent_id) REFERENCES discord_account_links (user_id),
    CONSTRAINT chk_discord_account_links_not_self
        CHECK (parent_id IS NULL OR parent_id <> user_id)
);

-- +goose Down
DROP TABLE discord_account_links;
