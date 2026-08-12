// Package migration owns the connection and Goose execution used to evolve
// the shared Salada schema.
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/kiLeo13/SaladaFun/database/internal/config"
	"github.com/pressly/goose/v3"
)

// Open creates and verifies the MySQL pool used exclusively by migrations.
func Open(ctx context.Context, configuration config.Config) (*sql.DB, error) {
	database, err := sql.Open("mysql", dataSourceName(configuration))
	if err != nil {
		return nil, fmt.Errorf("open migration database: %w", err)
	}
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping migration database: %w", err)
	}
	return database, nil
}

// Up applies every pending SQL migration.
func Up(ctx context.Context, database *sql.DB, path string) error {
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("configure goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, database, path); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	return nil
}

func dataSourceName(configuration config.Config) string {
	driverConfig := mysqldriver.NewConfig()
	driverConfig.User = configuration.Username
	driverConfig.Passwd = configuration.Password
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(configuration.Host, strconv.FormatUint(uint64(configuration.Port), 10))
	driverConfig.DBName = configuration.DatabaseName
	driverConfig.ParseTime = true
	driverConfig.MultiStatements = true
	driverConfig.Loc = time.UTC
	return driverConfig.FormatDSN()
}
