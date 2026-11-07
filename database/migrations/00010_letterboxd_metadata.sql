-- +goose Up
-- Unreleased feature: letterboxd_titles must be empty before applying this migration.
ALTER TABLE letterboxd_titles
 ADD COLUMN creators JSON NOT NULL,
 ADD COLUMN genres JSON NOT NULL,
 ADD COLUMN homepage VARCHAR(2048) NOT NULL,
 ADD COLUMN season_count INT UNSIGNED NOT NULL,
 ADD CONSTRAINT chk_letterboxd_creators_array CHECK (JSON_TYPE(creators) = 'ARRAY'),
 ADD CONSTRAINT chk_letterboxd_genres_array CHECK (JSON_TYPE(genres) = 'ARRAY');

-- +goose Down
ALTER TABLE letterboxd_titles
 DROP CHECK chk_letterboxd_creators_array,
 DROP CHECK chk_letterboxd_genres_array,
 DROP COLUMN creators,
 DROP COLUMN genres,
 DROP COLUMN homepage,
 DROP COLUMN season_count;
