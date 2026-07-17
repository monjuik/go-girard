package app

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenDatabase(t *testing.T) {
	db := openTestDatabase(t)

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("read journal mode: %v", err)
	}

	if journalMode != "wal" {
		t.Fatalf("journal mode = %q, want %q", journalMode, "wal")
	}
}

func TestOpenDatabaseWithRelativePath(t *testing.T) {
	t.Chdir(t.TempDir())

	db, err := OpenDatabase(context.Background(), "test.db")
	if err != nil {
		t.Fatalf("OpenDatabase() error = %v", err)
	}
	defer db.Close()
}

func TestMigrate(t *testing.T) {
	db := openTestDatabase(t)

	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// A second run must not apply the migration again.
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO company (id, name)
		VALUES (1, 'Northwind Logistics');

		INSERT INTO person (id, name, position, company)
		VALUES (101, 'Anna Petrova', 'Head of Operations', 1);
	`); err != nil {
		t.Fatalf("insert migrated schema data: %v", err)
	}

	var migrationCount int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM migration WHERE name = ?",
		"001_initial",
	).Scan(&migrationCount); err != nil {
		t.Fatalf("count migrations: %v", err)
	}

	if migrationCount != 1 {
		t.Fatalf("migration count = %d, want 1", migrationCount)
	}
}

func TestApplyMigrationRollsBackOnError(t *testing.T) {
	db := openTestDatabase(t)

	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	broken := migration{
		name: "test_broken",
		sql: `
			CREATE TABLE temporary_test (
				id INTEGER PRIMARY KEY
			);

			THIS IS NOT VALID SQL;
		`,
	}

	if err := applyMigration(context.Background(), db, broken); err == nil {
		t.Fatal("applyMigration() error = nil, want error")
	}

	var tableExists bool
	if err := db.QueryRow(
		`SELECT EXISTS(
			SELECT 1
			FROM sqlite_master
			WHERE type = 'table' AND name = 'temporary_test'
		)`,
	).Scan(&tableExists); err != nil {
		t.Fatalf("check temporary table: %v", err)
	}

	if tableExists {
		t.Fatal("temporary_test exists after failed migration")
	}

	var migrationCount int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM migration WHERE name = ?",
		broken.name,
	).Scan(&migrationCount); err != nil {
		t.Fatalf("count broken migration records: %v", err)
	}

	if migrationCount != 0 {
		t.Fatalf("broken migration count = %d, want 0", migrationCount)
	}
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	db, err := OpenDatabase(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	return db
}
