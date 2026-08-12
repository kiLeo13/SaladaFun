package database

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/config"
)

func TestOpenAgainstMySQL(t *testing.T) {
	configuration := liveConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := Open(ctx, configuration)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	pool, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	var count int64
	if err := database.Raw("SELECT 1").Scan(&count).Error; err != nil || count != 1 {
		t.Fatalf("GORM count = %d, error = %v", count, err)
	}
}

func TestSettingsBuildsDriverDSN(t *testing.T) {
	t.Parallel()
	configuration := config.DatabaseConfig{
		Host: "mysql.internal", Port: 3307, Username: "salada",
		Password: "p@ss:/word", Name: "salada",
	}
	parsed, err := mysqldriver.ParseDSN(dataSourceName(configuration))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Addr != "mysql.internal:3307" || parsed.User != "salada" || parsed.Passwd != "p@ss:/word" || parsed.DBName != "salada" || !parsed.ParseTime || parsed.MultiStatements || parsed.Loc != time.UTC {
		t.Fatalf("parsed DSN = %#v", parsed)
	}
}

func liveConfig(t *testing.T) config.DatabaseConfig {
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
	return config.DatabaseConfig{
		Host: values["host"], Port: uint16(port), Username: values["username"],
		Password: values["password"], Name: values["name"], MaxOpen: 3,
		MaxIdle: 1, MaxLifetime: time.Minute,
	}
}
