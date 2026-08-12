-- +goose Up
CREATE TABLE config (
    name VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (name)
);

-- +goose Down
DROP TABLE config;
