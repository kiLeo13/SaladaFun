-- +goose Up
CREATE TABLE config (
    name VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (name)
);

CREATE TABLE birthdays (
    user_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(100) NOT NULL,
    birthday DATE NOT NULL,
    time_zone VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    created_at BIGINT UNSIGNED NOT NULL,
    updated_at BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (user_id)
);

CREATE TABLE birthday_announcements (
    user_id BIGINT UNSIGNED NOT NULL,
    birthday_date DATE NOT NULL,
    sent_at BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (user_id, birthday_date),
    CONSTRAINT fk_birthday_announcements_user
        FOREIGN KEY (user_id) REFERENCES birthdays (user_id)
        ON DELETE CASCADE
);

-- +goose Down
DROP TABLE birthday_announcements;
DROP TABLE birthdays;
DROP TABLE config;
