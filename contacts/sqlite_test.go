package contacts_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"

	"github.com/monjuik/go-girard/app"
	"github.com/monjuik/go-girard/contacts"
)

func TestSQLitePersonQueries(t *testing.T) {
	db := openTestDatabase(t)
	insertPersonFixtures(t, db)

	queries := contacts.NewSQLitePersonQueries(db)

	tests := []struct {
		name      string
		filter    contacts.PersonsFilter
		wantNames []string
	}{
		{
			name:      "all persons",
			filter:    contacts.PersonsFilter{Limit: 20},
			wantNames: []string{"Anna Petrova", "Mark Jensen", "Zoe Miller"},
		},
		{
			name:      "name ignoring case",
			filter:    contacts.PersonsFilter{Query: "ANNA", Limit: 20},
			wantNames: []string{"Anna Petrova"},
		},
		{
			name:      "position ignoring case",
			filter:    contacts.PersonsFilter{Query: "founder", Limit: 20},
			wantNames: []string{"Mark Jensen"},
		},
		{
			name:      "company ignoring case",
			filter:    contacts.PersonsFilter{Query: "northWIND", Limit: 20},
			wantNames: []string{"Anna Petrova"},
		},
		{
			name:      "literal percent",
			filter:    contacts.PersonsFilter{Query: "%", Limit: 20},
			wantNames: []string{"Zoe Miller"},
		},
		{
			name:      "literal underscore",
			filter:    contacts.PersonsFilter{Query: "_", Limit: 20},
			wantNames: []string{"Zoe Miller"},
		},
		{
			name:      "no match",
			filter:    contacts.PersonsFilter{Query: "missing", Limit: 20},
			wantNames: nil,
		},
		{
			name:      "paging",
			filter:    contacts.PersonsFilter{Skip: 1, Limit: 1},
			wantNames: []string{"Mark Jensen"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := queries.ListPersonRows(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("ListPersonRows() error = %v", err)
			}

			gotNames := make([]string, len(rows))
			for i, row := range rows {
				gotNames[i] = row.Name
			}

			if !slices.Equal(gotNames, tt.wantNames) {
				t.Fatalf("ListPersonRows() names = %v, want %v", gotNames, tt.wantNames)
			}
		})
	}
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	db, err := app.OpenDatabase(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if err := app.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	return db
}

func insertPersonFixtures(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
 		INSERT INTO company (id, name) VALUES
			(1, 'Northwind Logistics'),
			(2, 'Acme_100%');

		INSERT INTO person (id, name, position, company) VALUES
			(101, 'Anna Petrova', 'Head of Operations', 1),
			(102, 'Mark Jensen', 'Founder', NULL),
			(103, 'Zoe Miller', 'Engineer', 2);
	`)
	if err != nil {
		t.Fatalf("insert fixtures: %v", err)
	}
}
