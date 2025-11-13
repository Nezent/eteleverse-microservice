package database

import (
	"database/sql"
	"fmt"
	"os"
	"runtime"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

// Database wraps the Bun DB connection and provides database access methods.
type Database struct {
	DB *bun.DB
}

// NewDatabase initializes and returns a new Database instance.
func NewDatabase() (*Database, error) {
	// Get database configuration from environment variables with defaults
	host := getEnv("POSTGRES_HOST", "db")
	user := getEnv("POSTGRES_USER", "nezent")
	password := getEnv("POSTGRES_PASSWORD", "123456")
	dbname := getEnv("POSTGRES_DB", "nezent_db")
	sslmode := getEnv("POSTGRES_SSL_MODE", "disable")

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", user, password, host, dbname, sslmode)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)

	db := bun.NewDB(sqldb, pgdialect.New())

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))
	return &Database{
		DB: db,
	}, nil
}

// Close closes the database connection.
func (db *Database) Close() error {
	return db.DB.DB.Close()
}

// RawSQLDB returns the underlying *sql.DB instance.
func (db *Database) RawSQLDB() *sql.DB {
	return db.DB.DB
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
