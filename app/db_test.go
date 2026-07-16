package app

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	db, err := OpenDatabase(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenDatabase() error = %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("read journal mode: %v", err)
	}

	if journalMode != "wal" {
		t.Fatalf("journal mode = %q, want %q", journalMode, "wal")
	}
}
