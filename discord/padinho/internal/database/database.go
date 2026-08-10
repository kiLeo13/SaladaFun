package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const schemaCharacterSet = "utf8mb4"

var (
	databaseNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	ErrDatabaseName     = errors.New("database name must be a MySQL identifier")
)

// Settings contains typed MySQL connection and pool configuration. Callers do
// not construct or pass driver-specific DSNs.
type Settings struct {
	Host        string
	Port        uint16
	Username    string
	Password    string
	Name        string
	MaxOpen     int
	MaxIdle     int
	MaxLifetime time.Duration
}

// Connection owns both GORM and its underlying database/sql pool.
type Connection struct {
	GORM *gorm.DB
	SQL  *sql.DB
}

// Open creates the configured schema when absent, then creates and verifies a
// bounded MySQL connection pool for it.
func Open(ctx context.Context, settings Settings) (*Connection, error) {
	if !databaseNamePattern.MatchString(settings.Name) {
		return nil, fmt.Errorf("%w: %q", ErrDatabaseName, settings.Name)
	}
	if err := ensureSchema(ctx, settings); err != nil {
		return nil, err
	}
	database, err := gorm.Open(
		gormmysql.Open(settings.dsn(settings.Name)),
		&gorm.Config{DisableAutomaticPing: true},
	)
	if err != nil {
		return nil, fmt.Errorf("open MySQL with GORM: %w", err)
	}
	pool, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("access MySQL connection pool: %w", err)
	}
	pool.SetMaxOpenConns(settings.MaxOpen)
	pool.SetMaxIdleConns(settings.MaxIdle)
	pool.SetConnMaxLifetime(settings.MaxLifetime)
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	return &Connection{GORM: database, SQL: pool}, nil
}

func ensureSchema(ctx context.Context, settings Settings) error {
	bootstrap, err := sql.Open("mysql", settings.dsn(""))
	if err != nil {
		return fmt.Errorf("open MySQL bootstrap connection: %w", err)
	}
	defer bootstrap.Close()
	if err := bootstrap.PingContext(ctx); err != nil {
		return fmt.Errorf("ping MySQL bootstrap connection: %w", err)
	}
	statement := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s",
		settings.Name, schemaCharacterSet,
	)
	if _, err := bootstrap.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("create MySQL database %s: %w", settings.Name, err)
	}
	return nil
}

func (s Settings) dsn(databaseName string) string {
	configuration := mysqldriver.NewConfig()
	configuration.User = s.Username
	configuration.Passwd = s.Password
	configuration.Net = "tcp"
	configuration.Addr = net.JoinHostPort(s.Host, strconv.FormatUint(uint64(s.Port), 10))
	configuration.DBName = databaseName
	configuration.ParseTime = true
	configuration.MultiStatements = true
	configuration.Loc = time.UTC
	return configuration.FormatDSN()
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
