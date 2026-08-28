package db

import (
	"context"
	"embed"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(ctx context.Context, path string) (*sqlx.DB, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(3000)"

	database, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := migrate(database); err != nil {
		database.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return database, nil
}

func migrate(database *sqlx.DB) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.Up(database.DB, "migrations"); err != nil {
		return err
	}

	return nil
}
