// Package database bootstraps Padinho's GORM connection from environment
// variables. Schema creation and migration belong to the root database module.
package database

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	defaultMaxOpen     = 5
	defaultMaxIdle     = 2
	defaultMaxLifetime = 30 * time.Minute
)

type settings struct {
	host        string
	port        uint16
	user        string
	password    string
	name        string
	maxOpen     int
	maxIdle     int
	maxLifetime time.Duration
}

// Open reads Padinho's DB_* variables and returns a verified GORM connection.
func Open() (*gorm.DB, error) {
	cfg, err := load()
	if err != nil {
		return nil, err
	}
	database, err := gorm.Open(
		gormmysql.Open(dataSourceName(cfg)),
		&gorm.Config{DisableAutomaticPing: true},
	)
	if err != nil {
		return nil, fmt.Errorf("open MySQL with GORM: %w", err)
	}
	pool, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("access MySQL connection pool: %w", err)
	}
	pool.SetMaxOpenConns(cfg.maxOpen)
	pool.SetMaxIdleConns(cfg.maxIdle)
	pool.SetConnMaxLifetime(cfg.maxLifetime)
	if err := pool.Ping(); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	return database, nil
}

// Close closes GORM's underlying connection pool.
func Close(database *gorm.DB) error {
	pool, err := database.DB()
	if err != nil {
		return fmt.Errorf("access MySQL connection pool: %w", err)
	}
	return pool.Close()
}

func load() (settings, error) {
	cfg := settings{
		host: os.Getenv("DB_HOST"), user: os.Getenv("DB_USER"),
		password: os.Getenv("DB_PASSWORD"), name: os.Getenv("DB_NAME"),
	}
	if cfg.host == "" {
		return settings{}, errors.New("DB_HOST is required")
	}
	var err error
	if cfg.port, err = port(); err != nil {
		return settings{}, err
	}
	if cfg.user == "" {
		return settings{}, errors.New("DB_USER is required")
	}
	if cfg.password == "" {
		return settings{}, errors.New("DB_PASSWORD is required")
	}
	if cfg.name == "" {
		return settings{}, errors.New("DB_NAME is required")
	}
	if cfg.maxOpen, err = positiveInt("DB_MAX_OPEN", defaultMaxOpen); err != nil {
		return settings{}, err
	}
	if cfg.maxIdle, err = positiveInt("DB_MAX_IDLE", defaultMaxIdle); err != nil {
		return settings{}, err
	}
	if cfg.maxIdle > cfg.maxOpen {
		return settings{}, errors.New("DB_MAX_IDLE cannot exceed DB_MAX_OPEN")
	}
	if cfg.maxLifetime, err = positiveDuration("DB_MAX_LIFETIME", defaultMaxLifetime); err != nil {
		return settings{}, err
	}
	return cfg, nil
}

func port() (uint16, error) {
	value := os.Getenv("DB_PORT")
	parsed, err := strconv.ParseUint(value, 10, 16)
	if value == "" || err != nil || parsed == 0 {
		return 0, errors.New("DB_PORT must be an integer between 1 and 65535")
	}
	return uint16(parsed), nil
}

func positiveInt(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}

func positiveDuration(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive Go duration", name)
	}
	return parsed, nil
}

func dataSourceName(cfg settings) string {
	driverConfig := mysqldriver.NewConfig()
	driverConfig.User = cfg.user
	driverConfig.Passwd = cfg.password
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(cfg.host, strconv.FormatUint(uint64(cfg.port), 10))
	driverConfig.DBName = cfg.name
	driverConfig.ParseTime = true
	driverConfig.Loc = time.UTC
	return driverConfig.FormatDSN()
}
