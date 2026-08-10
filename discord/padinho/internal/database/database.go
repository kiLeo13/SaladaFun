package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Connection owns both GORM and its underlying database/sql pool.
type Connection struct {
	GORM *gorm.DB
	SQL  *sql.DB
}

// Open creates and verifies a bounded MySQL connection pool.
func Open(ctx context.Context, dsn string, maxOpen, maxIdle int, maxLifetime time.Duration) (*Connection, error) {
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		return nil, fmt.Errorf("open MySQL with GORM: %w", err)
	}
	pool, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("access MySQL connection pool: %w", err)
	}
	pool.SetMaxOpenConns(maxOpen)
	pool.SetMaxIdleConns(maxIdle)
	pool.SetConnMaxLifetime(maxLifetime)
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	return &Connection{GORM: database, SQL: pool}, nil
}

// Migrate applies every pending SQL migration before Discord is connected.
func Migrate(ctx context.Context, database *sql.DB, path string) error {
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("configure goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, database, path); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	return nil
}
