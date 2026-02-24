package database

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds DB connection config (from env per DB_SPEC).
type Config struct {
	DSN string
}

// DefaultConfig reads DSN from env. Supports both PostgreSQL and Supabase:
// - DATABASE_URL: standard Postgres or Supabase connection string
// - SUPABASE_DB_URL: Supabase Database URL (if set, used when DATABASE_URL is empty)
// Supabase: use connection string from Dashboard → Project Settings → Database (sslmode=require).
func DefaultConfig() Config {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("SUPABASE_DB_URL")
	}
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=account_stock port=5432 sslmode=disable"
	}
	return Config{DSN: dsn}
}

var db *gorm.DB

// Open opens a PostgreSQL connection and sets the package-level db.
// Call once at startup; use DB() in handlers. Safe to call when DATABASE_URL is unset (uses dev default).
func Open(cfg Config) (*gorm.DB, error) {
	var err error
	db, err = gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // reduce log in prod; set Info in dev if needed
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	log.Println("database: connected")
	return db, nil
}

// DB returns the global GORM DB. Must call Open() first.
func DB() *gorm.DB {
	return db
}

// Close closes the underlying sql.DB. Call in defer after Open (e.g. in main).
func Close() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
