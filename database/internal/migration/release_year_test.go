package migration

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	migrationfiles "github.com/kiLeo13/SaladaFun/database/migrations"
	"github.com/pressly/goose/v3"
)

// TestUnknownReleaseYearAgainstMySQL verifies upgrade preservation, bounds and lossless rollback guards.
func TestUnknownReleaseYearAgainstMySQL(t *testing.T) {
	setLiveEnvironment(t)
	t.Setenv("DB_NAME", fmt.Sprintf("salada_unknown_year_%d", time.Now().UnixNano()))
	db, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	goose.SetBaseFS(migrationfiles.Files)
	if err := goose.SetDialect("mysql"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, migrationsPath, 11); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO letterboxd_titles
 (id, tmdb_id, media_type, original_title, localized_title, release_year,
 synopsis, poster_path, backdrop_path, accent_source, metadata_fetched_at,
 creators, genres, homepage, season_count, created_at, updated_at)
 VALUES (1, 1284679, 'movie', 'Planned', '', 2026, '', '', '', '', 999,
 JSON_ARRAY(), JSON_ARRAY(), '', 0, 1, 1)`); err != nil {
		t.Fatal(err)
	}
	if err := Up(db); err != nil {
		t.Fatal(err)
	}
	var year sql.NullInt64
	var fetched int64
	if err := db.QueryRow("SELECT release_year, metadata_fetched_at FROM letterboxd_titles WHERE id=1").Scan(&year, &fetched); err != nil || !year.Valid || year.Int64 != 2026 || fetched != 999 {
		t.Fatalf("preserve metadata: %v %d %v", year, fetched, err)
	}
	for _, value := range []string{"0", "10000"} {
		if _, err := db.Exec("UPDATE letterboxd_titles SET release_year=" + value + " WHERE id=1"); err == nil {
			t.Fatalf("invalid year accepted: %s", value)
		}
	}
	if _, err := db.Exec("UPDATE letterboxd_titles SET release_year=NULL WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Down(db, migrationsPath); err == nil {
		t.Fatal("rollback must reject unknown years")
	}
	if err := db.QueryRow("SELECT release_year FROM letterboxd_titles WHERE id=1").Scan(&year); err != nil || year.Valid {
		t.Fatalf("rollback changed unknown year: %v %v", year, err)
	}
	if _, err := db.Exec("UPDATE letterboxd_titles SET release_year=2027 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Down(db, migrationsPath); err != nil {
		t.Fatal("rollback with known years", err)
	}
	if _, err := db.Exec("UPDATE letterboxd_titles SET release_year=NULL WHERE id=1"); err == nil {
		t.Fatal("rollback did not restore NOT NULL")
	}
	if err := Up(db); err != nil {
		t.Fatal("reapply migration", err)
	}
}
