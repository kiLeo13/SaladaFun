-- +goose Up
-- Future metadata is nullable: NULL means not fetched or unknown, never zero/unrestricted.
-- No cache invalidation is needed until application code starts fetching these fields.
ALTER TABLE letterboxd_titles
 ADD COLUMN content_ratings JSON NULL,
 ADD COLUMN runtime_minutes SMALLINT UNSIGNED NULL,
 ADD COLUMN episode_runtime_minutes JSON NULL,
 ADD COLUMN original_language VARCHAR(10) NULL,
 ADD COLUMN origin_countries JSON NULL,
 ADD COLUMN status VARCHAR(64) NULL,
 ADD COLUMN episode_count INT UNSIGNED NULL,
 ADD COLUMN external_ids JSON NULL,
 ADD CONSTRAINT chk_letterboxd_content_ratings_array
  CHECK (content_ratings IS NULL OR JSON_TYPE(content_ratings) = 'ARRAY'),
 ADD CONSTRAINT chk_letterboxd_runtime_positive
  CHECK (runtime_minutes IS NULL OR runtime_minutes > 0),
 ADD CONSTRAINT chk_letterboxd_episode_runtime_array
  CHECK (episode_runtime_minutes IS NULL OR JSON_TYPE(episode_runtime_minutes) = 'ARRAY'),
 ADD CONSTRAINT chk_letterboxd_origin_countries_array
  CHECK (origin_countries IS NULL OR JSON_TYPE(origin_countries) = 'ARRAY'),
 ADD CONSTRAINT chk_letterboxd_external_ids_object
  CHECK (external_ids IS NULL OR JSON_TYPE(external_ids) = 'OBJECT');

-- +goose Down
ALTER TABLE letterboxd_titles
 DROP CHECK chk_letterboxd_content_ratings_array,
 DROP CHECK chk_letterboxd_runtime_positive,
 DROP CHECK chk_letterboxd_episode_runtime_array,
 DROP CHECK chk_letterboxd_origin_countries_array,
 DROP CHECK chk_letterboxd_external_ids_object,
 DROP COLUMN content_ratings,
 DROP COLUMN runtime_minutes,
 DROP COLUMN episode_runtime_minutes,
 DROP COLUMN original_language,
 DROP COLUMN origin_countries,
 DROP COLUMN status,
 DROP COLUMN episode_count,
 DROP COLUMN external_ids;
