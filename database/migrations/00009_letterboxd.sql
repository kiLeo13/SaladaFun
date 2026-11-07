-- +goose Up
CREATE TABLE letterboxd_titles (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 tmdb_id BIGINT UNSIGNED NOT NULL,
 media_type VARCHAR(5) NOT NULL,
 original_title VARCHAR(500) NOT NULL,
 localized_title VARCHAR(500) NOT NULL,
 release_year SMALLINT UNSIGNED NOT NULL,
 release_date DATE NULL,
 synopsis TEXT NOT NULL,
 poster_path VARCHAR(512) NOT NULL,
 backdrop_path VARCHAR(512) NOT NULL,
 accent INT UNSIGNED NULL,
 accent_source VARCHAR(512) NOT NULL,
 metadata_fetched_at BIGINT UNSIGNED NOT NULL,
 created_at BIGINT UNSIGNED NOT NULL,
 updated_at BIGINT UNSIGNED NOT NULL,
 UNIQUE KEY uq_letterboxd_identity (media_type, tmdb_id),
 CHECK (tmdb_id > 0 AND media_type IN ('movie', 'tv')),
 CHECK (CHAR_LENGTH(TRIM(original_title)) > 0),
 CHECK (release_year BETWEEN 1 AND 9999),
 CHECK (accent IS NULL OR accent <= 16777215)
);
CREATE TABLE letterboxd_wishlist_items (
 user_id BIGINT UNSIGNED NOT NULL,
 title_id BIGINT UNSIGNED NOT NULL,
 created_at BIGINT UNSIGNED NOT NULL,
 PRIMARY KEY (user_id, title_id),
 FOREIGN KEY (title_id) REFERENCES letterboxd_titles(id),
 CHECK (user_id > 0)
);
CREATE TABLE letterboxd_watches (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 user_id BIGINT UNSIGNED NOT NULL,
 title_id BIGINT UNSIGNED NOT NULL,
 watched_at BIGINT UNSIGNED NOT NULL,
 rating TINYINT UNSIGNED NOT NULL,
 mutation_id BIGINT UNSIGNED NOT NULL,
 created_at BIGINT UNSIGNED NOT NULL,
 updated_at BIGINT UNSIGNED NOT NULL,
 UNIQUE KEY uq_letterboxd_watch (user_id, title_id),
 KEY idx_letterboxd_title_rating (title_id, rating),
 FOREIGN KEY (title_id) REFERENCES letterboxd_titles(id),
 CHECK (user_id > 0 AND mutation_id > 0 AND rating BETWEEN 1 AND 5)
);
CREATE TABLE letterboxd_reviews (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 watch_id BIGINT UNSIGNED NOT NULL,
 review VARCHAR(500) NULL,
 interaction_id BIGINT UNSIGNED NOT NULL,
 created_at BIGINT UNSIGNED NOT NULL,
 UNIQUE KEY uq_letterboxd_review_interaction (interaction_id),
 KEY idx_letterboxd_revision (watch_id, id),
 FOREIGN KEY (watch_id) REFERENCES letterboxd_watches(id),
 CHECK (interaction_id > 0),
 CHECK (review IS NULL OR CHAR_LENGTH(TRIM(review)) BETWEEN 2 AND 500)
);
-- +goose Down
DROP TABLE letterboxd_reviews;
DROP TABLE letterboxd_watches;
DROP TABLE letterboxd_wishlist_items;
DROP TABLE letterboxd_titles;
