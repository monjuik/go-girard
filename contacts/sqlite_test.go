package contacts_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"

	"github.com/monjuik/go-girard/app"
	"github.com/monjuik/go-girard/common"
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

	person, err := queries.GetPerson(context.Background(), common.ID(101))
	if err != nil {
		t.Fatalf("GetPerson() error = %v", err)
	}

	wantPerson := contacts.PersonView{
		ID:       common.ID(101).String(),
		Name:     "Anna Petrova",
		Position: "Head of Operations",
	}
	if person != wantPerson {
		t.Fatalf("GetPerson() = %+v, want %+v", person, wantPerson)
	}

	for _, id := range []common.ID{104, 999} {
		_, err := queries.GetPerson(context.Background(), id)
		if !errors.Is(err, contacts.ErrPersonNotFound) {
			t.Fatalf(
				"GetPerson(%d) error = %v, want ErrPersonNotFound",
				id,
				err,
			)
		}
	}
}

func TestPersonServiceWithSQLite(t *testing.T) {
	db := openTestDatabase(t)
	repository := contacts.NewSQLitePersonRepository(db)
	commands := contacts.NewPersonService(repository)
	ctx := context.Background()

	annaID, err := commands.CreatePerson(ctx, contacts.PersonInput{
		Name:     "  Anna Petrova  ",
		Position: "  Engineer  ",
	})
	if err != nil {
		t.Fatalf("CreatePerson() error = %v", err)
	}
	assertPersonInput(
		t,
		db,
		annaID,
		contacts.PersonInput{
			Name:     "Anna Petrova",
			Position: "Engineer",
		},
	)

	_, err = commands.CreatePerson(
		ctx,
		contacts.PersonInput{Name: "anna petrova"},
	)
	if !errors.Is(err, contacts.ErrPersonNameExists) {
		t.Fatalf("duplicate CreatePerson() error = %v, want ErrPersonNameExists", err)
	}

	markID, err := commands.CreatePerson(
		ctx,
		contacts.PersonInput{Name: "Mark Jensen"},
	)
	if err != nil {
		t.Fatalf("CreatePerson(Mark) error = %v", err)
	}
	_, err = db.Exec(
		"INSERT INTO company (id, name) VALUES (1, 'Northwind Logistics')",
	)
	if err != nil {
		t.Fatalf("insert company: %v", err)
	}

	_, err = db.Exec(
		"UPDATE person SET company = 1 WHERE id = ?",
		annaID.Int64(),
	)
	if err != nil {
		t.Fatalf("assign company: %v", err)
	}

	err = commands.UpdatePerson(
		ctx,
		markID,
		contacts.PersonInput{Name: "ANNA PETROVA"},
	)
	if !errors.Is(err, contacts.ErrPersonNameExists) {
		t.Fatalf("duplicate UpdatePerson() error = %v, want ErrPersonNameExists", err)
	}

	err = commands.UpdatePerson(ctx, annaID, contacts.PersonInput{
		Name:     "  Alice Petrova  ",
		Position: "  Director  ",
	})
	if err != nil {
		t.Fatalf("UpdatePerson() error = %v", err)
	}
	assertPersonInput(
		t,
		db,
		annaID,
		contacts.PersonInput{
			Name:     "Alice Petrova",
			Position: "Director",
		},
	)

	var companyID sql.NullInt64
	err = db.QueryRow(
		"SELECT company FROM person WHERE id = ?",
		annaID.Int64(),
	).Scan(&companyID)
	if err != nil {
		t.Fatalf("query person company: %v", err)
	}

	if !companyID.Valid || companyID.Int64 != 1 {
		t.Fatalf("person company = %+v, want 1", companyID)
	}

	if _, err := db.Exec(
		"UPDATE person SET deleted = 1 WHERE id = ?",
		annaID.Int64(),
	); err != nil {
		t.Fatalf("mark person deleted: %v", err)
	}

	err = commands.UpdatePerson(
		ctx,
		annaID,
		contacts.PersonInput{Name: "New name"},
	)
	if !errors.Is(err, contacts.ErrPersonNotFound) {
		t.Fatalf("update deleted person error = %v, want ErrPersonNotFound", err)
	}

	_, err = commands.CreatePerson(
		ctx,
		contacts.PersonInput{Name: " \t "},
	)
	if !errors.Is(err, contacts.ErrPersonNameRequired) {
		t.Fatalf("empty CreatePerson() error = %v, want ErrPersonNameRequired", err)
	}

	_, err = commands.CreatePerson(
		ctx,
		contacts.PersonInput{Name: "alice petrova"},
	)
	if err != nil {
		t.Fatalf("reuse deleted person name: %v", err)
	}
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	// An SQLite in-memory database belongs to one connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

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

		INSERT INTO person (id, name, position, company, deleted) VALUES
			(101, 'Anna Petrova', 'Head of Operations', 1, 0),
			(102, 'Mark Jensen', 'Founder', NULL, 0),
			(103, 'Zoe Miller', 'Engineer', 2, 0),
			(104, 'Deleted Person', 'Former', NULL, 1);
	`)
	if err != nil {
		t.Fatalf("insert fixtures: %v", err)
	}
}

func assertPersonInput(
	t *testing.T,
	db *sql.DB,
	id common.ID,
	want contacts.PersonInput,
) {
	t.Helper()

	var got contacts.PersonInput
	err := db.QueryRow(
		"SELECT name, position FROM person WHERE id = ?",
		id.Int64(),
	).Scan(&got.Name, &got.Position)
	if err != nil {
		t.Fatalf("query person: %v", err)
	}

	if got != want {
		t.Fatalf("person = %+v, want %+v", got, want)
	}
}
