package migration

import (
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
	if len(files) == 0 {
		t.Fatal("migration executable contains no SQL files")
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
