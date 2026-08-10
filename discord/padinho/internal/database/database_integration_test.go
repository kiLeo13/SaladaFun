package database

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestOpenAndMigrateAgainstMySQL(t *testing.T) {
	settings := liveSettings(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connection, err := Open(ctx, settings)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.SQL.Close() })
	for run := 0; run < 2; run++ {
		if err := Migrate(ctx, connection.SQL, "testdata/migrations"); err != nil {
			t.Fatalf("Migrate() run %d error = %v", run+1, err)
		}
	}
	if err := connection.GORM.Exec("INSERT INTO migration_probe (name) VALUES (?)", "live-mysql").Error; err != nil {
		t.Fatalf("GORM insert error = %v", err)
	}
	var count int64
	if err := connection.GORM.Table("migration_probe").Where("name = ?", "live-mysql").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("GORM count = %d, error = %v", count, err)
	}
}

func TestSettingsBuildsDriverDSN(t *testing.T) {
	t.Parallel()
	settings := Settings{
		Host: "mysql.internal", Port: 3307, Username: "salada",
		Password: "p@ss:/word", Name: "salada",
	}
	parsed, err := mysqldriver.ParseDSN(settings.dsn(settings.Name))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Addr != "mysql.internal:3307" || parsed.User != "salada" || parsed.Passwd != "p@ss:/word" || parsed.DBName != "salada" || !parsed.ParseTime || !parsed.MultiStatements || parsed.Loc != time.UTC {
		t.Fatalf("parsed DSN = %#v", parsed)
	}
}

func TestOpenRejectsInvalidDatabaseName(t *testing.T) {
	t.Parallel()
	_, err := Open(context.Background(), Settings{Name: "salada-test"})
	if !errors.Is(err, ErrDatabaseName) {
		t.Fatalf("Open() error = %v", err)
	}
}

func liveSettings(t *testing.T) Settings {
	t.Helper()
	values := map[string]string{
		"host": os.Getenv("TEST_DATABASE_HOST"), "port": os.Getenv("TEST_DATABASE_PORT"),
		"username": os.Getenv("TEST_DATABASE_USERNAME"), "password": os.Getenv("TEST_DATABASE_PASSWORD"),
		"name": os.Getenv("TEST_DATABASE_NAME"),
	}
	for name, value := range values {
		if value == "" {
			t.Skipf("TEST_DATABASE_%s is not set", name)
		}
	}
	port, err := strconv.ParseUint(values["port"], 10, 16)
	if err != nil || port == 0 {
		t.Fatalf("TEST_DATABASE_PORT is invalid: %q", values["port"])
	}
	return Settings{
		Host: values["host"], Port: uint16(port), Username: values["username"],
		Password: values["password"], Name: values["name"], MaxOpen: 3,
		MaxIdle: 1, MaxLifetime: time.Minute,
	}
}
