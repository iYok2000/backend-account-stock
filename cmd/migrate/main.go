// Migrate runs versioned SQL migrations (Postgres or Supabase). Use from repo root:
//
//	go run ./cmd/migrate
//
// Reads DATABASE_URL or SUPABASE_DB_URL from env (see docs/DB_SPEC.md).
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("SUPABASE_DB_URL")
	}
	if dsn == "" {
		log.Fatal("set DATABASE_URL or SUPABASE_DB_URL")
	}

	// migrations path: relative to cwd (run from repo root)
	migrationsPath := "migrations"
	if p := os.Getenv("MIGRATIONS_PATH"); p != "" {
		migrationsPath = p
	}
	abs, err := filepath.Abs(migrationsPath)
	if err != nil {
		log.Fatalf("migrations path: %v", err)
	}
	sourceURL := "file://" + abs

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		log.Fatalf("migrate new: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate up: %v", err)
	}
	log.Println("migrate: up done")
}
