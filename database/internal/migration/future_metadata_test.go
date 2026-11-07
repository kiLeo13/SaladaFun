package migration

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	migrationfiles "github.com/kiLeo13/SaladaFun/database/migrations"
	"github.com/pressly/goose/v3"
)

// TestFutureMetadataAgainstMySQL verifies an additive upgrade, unknown defaults,
// JSON shapes, numeric constraints and compatibility with the current title writer.
func TestFutureMetadataAgainstMySQL(t *testing.T) {
	setLiveEnvironment(t)
	t.Setenv("DB_NAME", fmt.Sprintf("salada_future_metadata_%d", time.Now().UnixNano()))
	db, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	goose.SetBaseFS(migrationfiles.Files)
	if err := goose.SetDialect("mysql"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, migrationsPath, 10); err != nil {
		t.Fatal(err)
	}
	const insertTitle = `INSERT INTO letterboxd_titles
 (id, tmdb_id, media_type, original_title, localized_title, release_year,
 synopsis, poster_path, backdrop_path, accent, accent_source, metadata_fetched_at,
 creators, genres, homepage, season_count, created_at, updated_at)
 VALUES (?, ?, 'tv', 'Series', '', 2021, '', '', '/art.jpg', 1193046, '/art.jpg',
 999, JSON_ARRAY('Creator'), JSON_ARRAY('Drama'), 'https://example.test', 5, 1, 1)`
	if _, err := db.Exec(insertTitle, 1, 42); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"INSERT INTO letterboxd_wishlist_items (user_id, title_id, created_at) VALUES (11, 1, 1)",
		"INSERT INTO letterboxd_watches (id, user_id, title_id, watched_at, rating, mutation_id, created_at, updated_at) VALUES (1, 11, 1, 1, 5, 100, 1, 1)",
		"INSERT INTO letterboxd_reviews (watch_id, review, interaction_id, created_at) VALUES (1, 'Review', 100, 1)",
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := Up(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"letterboxd_titles", "letterboxd_wishlist_items", "letterboxd_watches", "letterboxd_reviews"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s rows: %d, %v", table, count, err)
		}
	}
	var creator, genre, homepage, source string
	var seasons, fetched, accent int
	if err := db.QueryRow(`SELECT JSON_UNQUOTE(JSON_EXTRACT(creators, '$[0]')),
 JSON_UNQUOTE(JSON_EXTRACT(genres, '$[0]')), homepage, season_count,
 metadata_fetched_at, accent, accent_source FROM letterboxd_titles WHERE id=1`).Scan(&creator, &genre, &homepage, &seasons, &fetched, &accent, &source); err != nil {
		t.Fatal(err)
	}
	if creator != "Creator" || genre != "Drama" || homepage != "https://example.test" || seasons != 5 || fetched != 999 || accent != 1193046 || source != "/art.jpg" {
		t.Fatal("existing metadata changed")
	}
	const futureColumns = "content_ratings, runtime_minutes, episode_runtime_minutes, original_language, origin_countries, status, episode_count, external_ids"
	var columns [8]sql.NullString
	if err := db.QueryRow("SELECT "+futureColumns+" FROM letterboxd_titles WHERE id=1").Scan(&columns[0], &columns[1], &columns[2], &columns[3], &columns[4], &columns[5], &columns[6], &columns[7]); err != nil {
		t.Fatal(err)
	}
	for i, column := range columns {
		if column.Valid {
			t.Fatalf("unknown field %d should be SQL NULL", i)
		}
	}
	// Current application inserts omit every new field and must keep working.
	if _, err := db.Exec(insertTitle, 2, 43); err != nil {
		t.Fatal("legacy insert", err)
	}
	if _, err := db.Exec(`UPDATE letterboxd_titles SET
 content_ratings = JSON_ARRAY(JSON_OBJECT('country', 'BR', 'certification', '14', 'release_type', 3)),
 runtime_minutes = 120, episode_runtime_minutes = JSON_ARRAY(45, 50),
 original_language = 'pt', origin_countries = JSON_ARRAY('BR', 'US'),
 status = 'Returning Series', episode_count = 42,
 external_ids = JSON_OBJECT('imdb', 'tt1234567', 'tvdb', '123') WHERE id=1`); err != nil {
		t.Fatal("valid metadata", err)
	}
	var rating, imdb string
	var runtime, episodeRuntime, episodes int
	if err := db.QueryRow(`SELECT
 JSON_UNQUOTE(JSON_EXTRACT(content_ratings, '$[0].certification')),
 JSON_UNQUOTE(JSON_EXTRACT(external_ids, '$.imdb')),
 runtime_minutes, JSON_EXTRACT(episode_runtime_minutes, '$[0]'), episode_count
 FROM letterboxd_titles WHERE id=1`).Scan(&rating, &imdb, &runtime, &episodeRuntime, &episodes); err != nil {
		t.Fatal(err)
	}
	if rating != "14" || imdb != "tt1234567" || runtime != 120 || episodeRuntime != 45 || episodes != 42 {
		t.Fatal("metadata round trip")
	}
	for _, assignment := range []string{
		"content_ratings = JSON_OBJECT()", "content_ratings = JSON_QUOTE('14')",
		"episode_runtime_minutes = JSON_OBJECT()", "origin_countries = JSON_QUOTE('BR')",
		"external_ids = JSON_ARRAY()", "external_ids = JSON_QUOTE('tt123')",
		"runtime_minutes = 0", "runtime_minutes = -1", "episode_count = -1",
	} {
		if _, err := db.Exec("UPDATE letterboxd_titles SET " + assignment + " WHERE id=1"); err == nil {
			t.Fatalf("invalid metadata accepted: %s", assignment)
		}
	}
	if _, err := db.Exec(`UPDATE letterboxd_titles SET content_ratings = JSON_ARRAY(),
 episode_runtime_minutes = JSON_ARRAY(), origin_countries = JSON_ARRAY(),
 external_ids = JSON_OBJECT(), episode_count = 0 WHERE id=1`); err != nil {
		t.Fatal("known empty metadata", err)
	}
}
