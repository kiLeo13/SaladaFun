package database

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/config"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Open creates and verifies Padinho's bounded GORM connection pool. Schema
// creation and migration are deliberately owned by the root database project.
func Open(ctx context.Context, configuration config.DatabaseConfig) (*gorm.DB, error) {
	database, err := gorm.Open(
		gormmysql.Open(dataSourceName(configuration)),
		&gorm.Config{DisableAutomaticPing: true},
	)
	if err != nil {
		return nil, fmt.Errorf("open MySQL with GORM: %w", err)
	}
	pool, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("access MySQL connection pool: %w", err)
	}
	pool.SetMaxOpenConns(configuration.MaxOpen)
	pool.SetMaxIdleConns(configuration.MaxIdle)
	pool.SetConnMaxLifetime(configuration.MaxLifetime)
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	return database, nil
}

func dataSourceName(configuration config.DatabaseConfig) string {
	driverConfig := mysqldriver.NewConfig()
	driverConfig.User = configuration.Username
	driverConfig.Passwd = configuration.Password
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(configuration.Host, strconv.FormatUint(uint64(configuration.Port), 10))
	driverConfig.DBName = configuration.Name
	driverConfig.ParseTime = true
	driverConfig.Loc = time.UTC
	return driverConfig.FormatDSN()
}
