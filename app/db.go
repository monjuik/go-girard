package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
)

const databaseBusyTimeout = 2000

func OpenDatabase(ctx context.Context, path string) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("database path cannot be empty")
	}
	dsn := buildDSN(path)
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

func buildDSN(path string) string {
	dsn := &url.URL{
		Scheme: "file",
		Path:   path,
	}
	query := dsn.Query()
	query.Set("_busy_timeout", fmt.Sprintf("%d", databaseBusyTimeout))
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_synchronous", "NORMAL")
	dsn.RawQuery = query.Encode()

	return dsn.String()
}
