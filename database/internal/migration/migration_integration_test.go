package migration

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/kiLeo13/SaladaFun/database/internal/config"
)

func TestUpAgainstMySQL(t *testing.T) {
	configuration := liveConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := Open(ctx, configuration)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	for run := 0; run < 2; run++ {
		if err := Up(ctx, database, migrationsPath(t)); err != nil {
			t.Fatalf("Up() run %d error = %v", run+1, err)
		}
	}
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin config probe: %v", err)
	}
	t.Cleanup(func() { _ = transaction.Rollback() })
	if _, err := transaction.ExecContext(ctx, "INSERT INTO config (name, value) VALUES (?, ?)", "test.migration.probe", "token"); err != nil {
		t.Fatalf("insert config: %v", err)
	}
	var value string
	if err := transaction.QueryRowContext(ctx, "SELECT value FROM config WHERE name = ?", "test.migration.probe").Scan(&value); err != nil || value != "token" {
		t.Fatalf("value = %q, error = %v", value, err)
	}
}

func TestDataSourceName(t *testing.T) {
	t.Parallel()
	configuration := config.Config{
		Host: "mysql.internal", Port: 3307, Username: "salada",
		Password: "p@ss:/word", DatabaseName: "salada",
	}
	parsed, err := mysqldriver.ParseDSN(dataSourceName(configuration))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Addr != "mysql.internal:3307" || parsed.User != "salada" || parsed.Passwd != "p@ss:/word" || parsed.DBName != "salada" || !parsed.ParseTime || !parsed.MultiStatements || parsed.Loc != time.UTC {
		t.Fatalf("parsed DSN = %#v", parsed)
	}
}

func liveConfig(t *testing.T) config.Config {
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
	return config.Config{
		Host: values["host"], Port: uint16(port), Username: values["username"],
		Password: values["password"], DatabaseName: values["name"],
	}
}

func migrationsPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve migration test path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}
