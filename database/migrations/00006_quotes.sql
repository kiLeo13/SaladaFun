-- +goose Up
CREATE TABLE quote_authors (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    discord_user_id BIGINT UNSIGNED NULL,
    created_at BIGINT UNSIGNED NOT NULL,
    updated_at BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT uq_quote_authors_name UNIQUE (name),
    CONSTRAINT uq_quote_authors_discord_user_id UNIQUE (discord_user_id),
    CONSTRAINT chk_quote_authors_name_not_blank CHECK (CHAR_LENGTH(TRIM(name)) > 0)
);

CREATE TABLE quotes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    author_id BIGINT UNSIGNED NOT NULL,
    original_quote VARCHAR(1800) NOT NULL,
    translated_quote VARCHAR(1800) NULL,
    source_url VARCHAR(2048) NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at BIGINT UNSIGNED NOT NULL,
    updated_at BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    KEY idx_quotes_enabled (enabled),
    CONSTRAINT fk_quotes_author
        FOREIGN KEY (author_id) REFERENCES quote_authors (id)
        ON DELETE RESTRICT,
    CONSTRAINT chk_quotes_original_quote_not_blank
        CHECK (CHAR_LENGTH(TRIM(original_quote)) > 0),
    CONSTRAINT chk_quotes_translated_quote_not_blank
        CHECK (translated_quote IS NULL OR CHAR_LENGTH(TRIM(translated_quote)) > 0)
);

-- +goose Down
DROP TABLE quotes;
DROP TABLE quote_authors;
