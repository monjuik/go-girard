package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const databaseBusyTimeout = 2000

type migration struct {
	name string
	sql  string
}

func OpenDatabase(ctx context.Context, path string) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("database path cannot be empty")
	}
	dsn, err := buildDSN(path)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// The application uses a single connection
	// to keep things predictable
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

func buildDSN(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve database path: %w", err)
	}

	dsn := &url.URL{
		Scheme: "file",
		Path:   absolutePath,
	}

	query := dsn.Query()
	query.Set("_busy_timeout", fmt.Sprintf("%d", databaseBusyTimeout))
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_synchronous", "NORMAL")
	dsn.RawQuery = query.Encode()

	return dsn.String(), nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS migration (
				name     TEXT PRIMARY KEY,
				executed INTEGER NOT NULL
			)
		`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	for _, m := range migrations {
		if err := applyMigration(ctx, db, m); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, migration migration) error {
	var applied bool
	if err := db.QueryRowContext(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM migration WHERE name = ?)",
		migration.name,
	).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", migration.name, err)
	}
	if applied {
		return nil
	}
	slog.Info("applying database migration", "name", migration.name)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %q: %w", migration.name, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migration.sql); err != nil {
		return fmt.Errorf("execute migration %q: %w", migration.name, err)
	}

	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO migration (name, executed) VALUES (?, ?)",
		migration.name,
		time.Now().UnixMilli(),
	); err != nil {
		return fmt.Errorf("record migration %q: %w", migration.name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %q: %w", migration.name, err)
	}
	slog.Info("applied database migration", "name", migration.name)

	return nil
}

var migrations = []migration{
	{
		name: "001_initial",
		sql: `
			CREATE TABLE company (
				id   INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				country TEXT NOT NULL DEFAULT '',
				deleted  INTEGER NOT NULL DEFAULT 0
						CHECK (deleted IN (0, 1))
			);
			CREATE TABLE person (
				id       INTEGER PRIMARY KEY,
				name     TEXT NOT NULL,
				position TEXT NOT NULL,
				company  INTEGER REFERENCES company(id),
				deleted  INTEGER NOT NULL DEFAULT 0
						CHECK (deleted IN (0, 1))
			);

			CREATE UNIQUE INDEX person_name
			ON person (name COLLATE NOCASE)
			WHERE deleted = 0;

			CREATE UNIQUE INDEX company_name
			ON company (name COLLATE NOCASE)
			WHERE deleted = 0;
		`,
	},
}
