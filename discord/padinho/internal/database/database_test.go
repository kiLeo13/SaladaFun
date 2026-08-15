package database

import (
	"os"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestLoad(t *testing.T) {
	setRequiredEnvironment(t)
	configuration, err := load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.host != "mysql.internal" || configuration.port != 3306 || configuration.user != "salada" || configuration.password != "secret" || configuration.name != "salada" || configuration.maxOpen != defaultMaxOpen || configuration.maxIdle != defaultMaxIdle || configuration.maxLifetime != defaultMaxLifetime {
		t.Fatalf("load() = %#v", configuration)
	}
}

func TestLoadOverridesPoolSettings(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("DB_MAX_OPEN", "7")
	t.Setenv("DB_MAX_IDLE", "3")
	t.Setenv("DB_MAX_LIFETIME", "45m")
	configuration, err := load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.maxOpen != 7 || configuration.maxIdle != 3 || configuration.maxLifetime != 45*time.Minute {
		t.Fatalf("load() = %#v", configuration)
	}
}

func TestLoadValidation(t *testing.T) {
	tests := map[string]func(*testing.T){
		"host":     func(t *testing.T) { t.Setenv("DB_HOST", "") },
		"port":     func(t *testing.T) { t.Setenv("DB_PORT", "70000") },
		"user":     func(t *testing.T) { t.Setenv("DB_USER", "") },
		"password": func(t *testing.T) { t.Setenv("DB_PASSWORD", "") },
		"name":     func(t *testing.T) { t.Setenv("DB_NAME", "") },
		"max open": func(t *testing.T) { t.Setenv("DB_MAX_OPEN", "0") },
		"max idle": func(t *testing.T) { t.Setenv("DB_MAX_IDLE", "0") },
		"idle exceeds open": func(t *testing.T) {
			t.Setenv("DB_MAX_OPEN", "1")
			t.Setenv("DB_MAX_IDLE", "2")
		},
		"lifetime": func(t *testing.T) { t.Setenv("DB_MAX_LIFETIME", "soon") },
	}
	for name, configure := range tests {
		t.Run(name, func(t *testing.T) {
			setRequiredEnvironment(t)
			configure(t)
			if _, err := load(); err == nil {
				t.Fatal("load() error = nil")
			}
		})
	}
}

func TestDataSourceName(t *testing.T) {
	configuration := settings{
		host: "mysql.internal", port: 3307, user: "salada",
		password: "p@ss:/word", name: "salada",
	}
	parsed, err := mysqldriver.ParseDSN(dataSourceName(configuration))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Addr != "mysql.internal:3307" || parsed.User != "salada" || parsed.Passwd != "p@ss:/word" || parsed.DBName != "salada" || !parsed.ParseTime || parsed.MultiStatements || parsed.Loc != time.UTC {
		t.Fatalf("parsed DSN = %#v", parsed)
	}
}

func TestOpenAgainstMySQL(t *testing.T) {
	setLiveEnvironment(t)
	database, err := Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = Close(database) })
	var result int
	if err := database.Raw("SELECT 1").Scan(&result).Error; err != nil || result != 1 {
		t.Fatalf("SELECT 1 = %d, %v", result, err)
	}
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DB_HOST", "mysql.internal")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_USER", "salada")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "salada")
}

func setLiveEnvironment(t *testing.T) {
	t.Helper()
	variables := map[string]string{
		"DB_HOST": "TEST_DATABASE_HOST", "DB_PORT": "TEST_DATABASE_PORT",
		"DB_USER": "TEST_DATABASE_USERNAME", "DB_PASSWORD": "TEST_DATABASE_PASSWORD",
		"DB_NAME": "TEST_DATABASE_NAME",
	}
	for target, source := range variables {
		value := os.Getenv(source)
		if value == "" {
			t.Skipf("%s is not set", source)
		}
		t.Setenv(target, value)
	}
}
