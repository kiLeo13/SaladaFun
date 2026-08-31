package migration

import (
	"database/sql"
	"io/fs"
	"os"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	migrationfiles "github.com/kiLeo13/SaladaFun/database/migrations"
)

func TestEmbeddedMigrations(t *testing.T) {
	files, err := fs.Glob(migrationfiles.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 7 {
		t.Fatalf("embedded migrations = %d, want 7", len(files))
	}
}

func TestLoad(t *testing.T) {
	setRequiredEnvironment(t)
	configuration, err := load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.host != "mysql.internal" || configuration.port != 3306 || configuration.user != "salada" || configuration.password != "secret" || configuration.name != "salada" {
		t.Fatalf("load() = %#v", configuration)
	}
}

func TestLoadValidation(t *testing.T) {
	tests := map[string]func(*testing.T){
		"host":     func(t *testing.T) { t.Setenv("DB_HOST", "") },
		"port":     func(t *testing.T) { t.Setenv("DB_PORT", "70000") },
		"user":     func(t *testing.T) { t.Setenv("DB_USER", "") },
		"password": func(t *testing.T) { t.Setenv("DB_PASSWORD", "") },
		"name":     func(t *testing.T) { t.Setenv("DB_NAME", "") },
		"name format": func(t *testing.T) {
			t.Setenv("DB_NAME", "Salada; DROP DATABASE salada")
		},
	}
	for name, configure := range tests {
		t.Run(name, func(t *testing.T) {
			setRequiredEnvironment(t)
			configure(t)
			if _, err := load(); err == nil {
				t.Fatal("load() error = nil")
			}
		})
	}
}

func TestDataSourceName(t *testing.T) {
	configuration := settings{
		host: "mysql.internal", port: 3307, user: "salada",
		password: "p@ss:/word", name: "salada",
	}
	parsed, err := mysqldriver.ParseDSN(dataSourceName(configuration))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Addr != "mysql.internal:3307" || parsed.User != "salada" || parsed.Passwd != "p@ss:/word" || parsed.DBName != "salada" || !parsed.ParseTime || !parsed.MultiStatements || parsed.Loc != time.UTC {
		t.Fatalf("parsed DSN = %#v", parsed)
	}
}

func TestUpAgainstMySQL(t *testing.T) {
	setLiveEnvironment(t)
	database, err := Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	for run := 0; run < 2; run++ {
		if err := Up(database); err != nil {
			t.Fatalf("up() run %d error = %v", run+1, err)
		}
	}
	transaction, err := database.Begin()
	if err != nil {
		t.Fatalf("begin config probe: %v", err)
	}
	t.Cleanup(func() { _ = transaction.Rollback() })
	if _, err := transaction.Exec("INSERT INTO config (name, value) VALUES (?, ?)", "test.migration.probe", "token"); err != nil {
		t.Fatalf("insert config: %v", err)
	}
	var value string
	if err := transaction.QueryRow("SELECT value FROM config WHERE name = ?", "test.migration.probe").Scan(&value); err != nil || value != "token" {
		t.Fatalf("value = %q, error = %v", value, err)
	}
	const userID uint64 = 900000000000000001
	if _, err := transaction.Exec(`
		INSERT INTO birthdays
			(user_id, name, birthday, time_zone, message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, "Leo", "2000-03-04", "America/Sao_Paulo", "Olá", 1, 1,
	); err != nil {
		t.Fatalf("insert birthday: %v", err)
	}
	if _, err := transaction.Exec(`
		INSERT INTO birthday_announcements (user_id, birthday_date, sent_at)
		VALUES (?, ?, ?)`, userID, "2026-03-04", 2,
	); err != nil {
		t.Fatalf("insert birthday announcement: %v", err)
	}
	var announcements int
	if err := transaction.QueryRow(
		"SELECT COUNT(*) FROM birthday_announcements WHERE user_id = ?", userID,
	).Scan(&announcements); err != nil || announcements != 1 {
		t.Fatalf("birthday announcements = %d, error = %v", announcements, err)
	}
	if _, err := transaction.Exec(`
		INSERT INTO users_preferences
			(user_id, auto_mudae_oc, auto_mudae_oq, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, userID, nil, nil, 3, 3,
	); err != nil {
		t.Fatalf("insert nullable user preferences: %v", err)
	}
	var autoMudaeOC sql.NullBool
	if err := transaction.QueryRow(
		"SELECT auto_mudae_oc FROM users_preferences WHERE user_id = ?", userID,
	).Scan(&autoMudaeOC); err != nil || autoMudaeOC.Valid {
		t.Fatalf("nullable auto_mudae_oc = %#v, error = %v", autoMudaeOC, err)
	}
	var autoMudaeOQ sql.NullBool
	if err := transaction.QueryRow("SELECT auto_mudae_oq FROM users_preferences WHERE user_id = ?", userID).Scan(&autoMudaeOQ); err != nil || autoMudaeOQ.Valid {
		t.Fatalf("nullable auto_mudae_oq = %#v, error = %v", autoMudaeOQ, err)
	}
	var ouroharvestColumnCount int
	if err := transaction.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
			AND table_name = 'users_preferences'
			AND column_name = 'auto_mudae_oh'`,
	).Scan(&ouroharvestColumnCount); err != nil || ouroharvestColumnCount != 0 {
		t.Fatalf("auto_mudae_oh columns = %d, error = %v", ouroharvestColumnCount, err)
	}
	result, err := transaction.Exec(`
		INSERT INTO quote_authors (name, discord_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, "John Lennon", nil, 4, 4,
	)
	if err != nil {
		t.Fatalf("insert quote author: %v", err)
	}
	authorID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("quote author ID: %v", err)
	}
	if _, err := transaction.Exec(`
		INSERT INTO quotes
			(author_id, original_quote, translated_quote, source_url, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		authorID, "Time you enjoy wasting, was not wasted.",
		"O tempo que você gosta de desperdiçar não foi desperdiçado.",
		"https://example.test/quotes/190", true, 5, 5,
	); err != nil {
		t.Fatalf("insert quote: %v", err)
	}
	var quoteAuthor string
	var translated sql.NullString
	if err := transaction.QueryRow(`
		SELECT quote_authors.name, quotes.translated_quote
		FROM quotes
		JOIN quote_authors ON quote_authors.id = quotes.author_id
		WHERE quote_authors.id = ?`, authorID,
	).Scan(&quoteAuthor, &translated); err != nil || quoteAuthor != "John Lennon" || !translated.Valid {
		t.Fatalf("stored quote = %q, %#v, error = %v", quoteAuthor, translated, err)
	}
	if _, err := transaction.Exec(`
		INSERT INTO quote_authors (name, discord_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, "John Lennon", nil, 6, 6,
	); err == nil {
		t.Fatal("duplicate quote author name was accepted")
	}
	if _, err := transaction.Exec(`
		INSERT INTO quote_authors (name, discord_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, "Padinho", userID, 7, 7,
	); err != nil {
		t.Fatalf("insert Discord quote author: %v", err)
	}
	if _, err := transaction.Exec(`
		INSERT INTO quote_authors (name, discord_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, "Padinho duplicate", userID, 8, 8,
	); err == nil {
		t.Fatal("duplicate Discord quote author was accepted")
	}
	if _, err := transaction.Exec(`
		INSERT INTO quotes
			(author_id, original_quote, translated_quote, source_url, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uint64(0), "Orphan quote", nil, nil, false, 9, 9,
	); err == nil {
		t.Fatal("orphan quote was accepted")
	}
	const rootAccountID uint64 = 900000000000000010
	result, err = transaction.Exec(`
		INSERT INTO discord_account_links
			(user_id, parent_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, rootAccountID, nil, 10, 10,
	)
	if err != nil {
		t.Fatalf("insert root account link: %v", err)
	}
	if linkID, linkIDErr := result.LastInsertId(); linkIDErr != nil || linkID <= 0 {
		t.Fatalf("root account link ID = %d, error = %v", linkID, linkIDErr)
	}
	const childAccountID uint64 = 900000000000000011
	if _, err := transaction.Exec(`
		INSERT INTO discord_account_links
			(user_id, parent_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, childAccountID, rootAccountID, 11, 11,
	); err != nil {
		t.Fatalf("insert child account link: %v", err)
	}
	const grandchildAccountID uint64 = 900000000000000012
	if _, err := transaction.Exec(`
		INSERT INTO discord_account_links
			(user_id, parent_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, grandchildAccountID, childAccountID, 12, 12,
	); err != nil {
		t.Fatalf("insert grandchild account link: %v", err)
	}
	if _, err := transaction.Exec(`
		INSERT INTO discord_account_links
			(user_id, parent_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, childAccountID, nil, 13, 13,
	); err == nil {
		t.Fatal("duplicate linked user was accepted")
	}
	if _, err := transaction.Exec(`
		INSERT INTO discord_account_links
			(user_id, parent_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, uint64(900000000000000013), uint64(900000000000000099), 14, 14,
	); err == nil {
		t.Fatal("linked user with a missing parent was accepted")
	}
	const selfParentedAccountID uint64 = 900000000000000014
	if _, err := transaction.Exec(`
		INSERT INTO discord_account_links
			(user_id, parent_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, selfParentedAccountID, selfParentedAccountID, 15, 15,
	); err == nil {
		t.Fatal("self-parented linked user was accepted")
	}
	if _, err := transaction.Exec(
		"DELETE FROM discord_account_links WHERE user_id = ?", rootAccountID,
	); err == nil {
		t.Fatal("linked user with descendants was deleted")
	}
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DB_HOST", "mysql.internal")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_USER", "salada")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "salada")
}

func setLiveEnvironment(t *testing.T) {
	t.Helper()
	variables := map[string]string{
		"DB_HOST": "TEST_DATABASE_HOST", "DB_PORT": "TEST_DATABASE_PORT",
		"DB_USER": "TEST_DATABASE_USERNAME", "DB_PASSWORD": "TEST_DATABASE_PASSWORD",
		"DB_NAME": "TEST_DATABASE_NAME",
	}
	for target, source := range variables {
		value := os.Getenv(source)
		if value == "" {
			t.Skipf("%s is not set", source)
		}
		t.Setenv(target, value)
	}
}
