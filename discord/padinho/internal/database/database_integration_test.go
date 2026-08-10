package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOpenAndMigrateAgainstMySQL(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connection, err := Open(ctx, dsn, 3, 1, time.Minute)
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
