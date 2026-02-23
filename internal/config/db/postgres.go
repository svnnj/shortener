package db

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func runMigrations(dsn string) error {
	slog.Info("running migration...")
	m, err := migrate.New("file://migrations", dsn)
	defer m.Close()
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}

type PostgresDB interface {
	PingDB() error
}

func NewPostgresDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open connection: %w", err)
	}

	if err = runMigrations(dsn); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migration step failed: %w", err)
	}

	return db, nil
}
