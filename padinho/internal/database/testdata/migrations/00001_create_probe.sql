-- +goose Up
CREATE TABLE migration_probe (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(64) NOT NULL,
    PRIMARY KEY (id)
);

-- +goose Down
DROP TABLE migration_probe;
