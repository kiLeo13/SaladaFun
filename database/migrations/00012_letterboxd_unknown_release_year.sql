-- +goose Up
-- The existing BETWEEN check accepts NULL and still rejects years outside 1..9999.
-- Preserve all cached values and freshness timestamps; operators control refresh.
ALTER TABLE letterboxd_titles MODIFY COLUMN release_year SMALLINT UNSIGNED NULL;

-- +goose Down
-- Refuse rollback while unknown years exist; never fabricate a year or delete a title.
ALTER TABLE letterboxd_titles
 ADD CONSTRAINT chk_letterboxd_release_year_known CHECK (release_year IS NOT NULL);
ALTER TABLE letterboxd_titles
 MODIFY COLUMN release_year SMALLINT UNSIGNED NOT NULL,
 DROP CHECK chk_letterboxd_release_year_known;
