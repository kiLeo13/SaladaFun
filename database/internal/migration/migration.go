// Package migration owns the connection and Goose execution used to evolve
// the shared Salada schema.
package migration

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
)

const migrationsPath = "/app/migrations"

type settings struct {
	host     string
	port     uint16
	user     string
	password string
	name     string
}

// Open reads the DB_* variables and returns a verified migration connection.
func Open() (*sql.DB, error) {
	cfg, err := load()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dataSourceName(cfg))
	if err != nil {
		return nil, fmt.Errorf("open migration database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping migration database: %w", err)
	}
	return db, nil
}

// Up applies every pending SQL migration from the packaged migration path.
func Up(db *sql.DB) error {
	return up(db, migrationsPath)
}

func up(db *sql.DB, path string) error {
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("configure goose dialect: %w", err)
	}
	if err := goose.Up(db, path); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	return nil
}

func load() (settings, error) {
	cfg := settings{
		host: os.Getenv("DB_HOST"), user: os.Getenv("DB_USER"),
		password: os.Getenv("DB_PASSWORD"), name: os.Getenv("DB_NAME"),
	}
	if cfg.host == "" {
		return settings{}, errors.New("DB_HOST is required")
	}
	value := os.Getenv("DB_PORT")
	port, err := strconv.ParseUint(value, 10, 16)
	if value == "" || err != nil || port == 0 {
		return settings{}, errors.New("DB_PORT must be an integer between 1 and 65535")
	}
	cfg.port = uint16(port)
	if cfg.user == "" {
		return settings{}, errors.New("DB_USER is required")
	}
	if cfg.password == "" {
		return settings{}, errors.New("DB_PASSWORD is required")
	}
	if cfg.name == "" {
		return settings{}, errors.New("DB_NAME is required")
	}
	return cfg, nil
}

func dataSourceName(cfg settings) string {
	driverConfig := mysqldriver.NewConfig()
	driverConfig.User = cfg.user
	driverConfig.Passwd = cfg.password
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(cfg.host, strconv.FormatUint(uint64(cfg.port), 10))
	driverConfig.DBName = cfg.name
	driverConfig.ParseTime = true
	driverConfig.MultiStatements = true
	driverConfig.Loc = time.UTC
	return driverConfig.FormatDSN()
}
