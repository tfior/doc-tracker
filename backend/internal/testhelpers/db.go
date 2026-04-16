// Package testhelpers provides shared utilities for integration tests.
// Tests that use OpenTestDB require a running PostgreSQL instance.
// By default they connect to a database named "doctracker_test"; override
// with the TEST_DB_NAME environment variable.
//
// Create the test database once before running tests:
//
//	make test-db
package testhelpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func dbConfig() (host, port, user, password, dbName string) {
	host = getenv("DB_HOST", "localhost")
	port = getenv("DB_PORT", "5432")
	user = getenv("DB_USER", "doctracker")
	password = getenv("DB_PASSWORD", "changeme")
	dbName = getenv("TEST_DB_NAME", "doctracker_test")
	return
}

func dsn(host, port, user, password, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbName)
}

// migrationsDir returns the absolute path to db/migrations, resolved
// relative to this source file so it works from any test package.
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	// file = .../backend/internal/testhelpers/db.go
	// go up two levels to reach backend/, then into db/migrations
	return filepath.Join(filepath.Dir(file), "../../db/migrations")
}

// OpenTestDB connects to the test database, runs any pending migrations,
// and registers a cleanup that closes the connection when the test ends.
// It fails the test immediately if the database is unreachable.
func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	host, port, user, password, dbName := dbConfig()
	createTestDBIfNotExists(t, host, port, user, password, dbName)

	d := dsn(host, port, user, password, dbName)
	db, err := sql.Open("postgres", d)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("cannot reach test database — run 'make dev' before 'make test': %v", err)
	}

	m, err := migrate.New("file://"+migrationsDir(), d)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("run migrations: %v", err)
	}
	m.Close()

	t.Cleanup(func() { db.Close() })
	return db
}

// TruncateUsers removes all rows from the users table (cascading to
// activity_logs) so each test starts with a clean slate.
func TruncateUsers(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("TRUNCATE TABLE users CASCADE"); err != nil {
		t.Fatalf("truncate users: %v", err)
	}
}

// createTestDBIfNotExists connects to the postgres system database and
// creates the test database if it does not already exist.
func createTestDBIfNotExists(t *testing.T, host, port, user, password, dbName string) {
	t.Helper()
	adminDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		user, password, host, port)
	db, err := sql.Open("postgres", adminDSN)
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE " + dbName)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("create test db: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
