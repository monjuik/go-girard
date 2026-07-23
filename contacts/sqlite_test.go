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

func TestSQLiteCompanyQueries(t *testing.T) {
	db := openTestDatabase(t)

	if _, err := db.Exec(`
		INSERT INTO company (id, name, country, deleted) VALUES
			(1, 'Northwind Logistics', 'Cyprus', 0),
			(2, 'Acme_100%', 'Denmark', 0),
			(3, 'Deleted Company', 'France', 1);
	`); err != nil {
		t.Fatalf("insert company fixtures: %v", err)
	}

	queries := contacts.NewSQLiteCompanyQueries(db)

	tests := []struct {
		name      string
		filter    contacts.CompaniesFilter
		wantNames []string
	}{
		{
			name:      "all companies",
			filter:    contacts.CompaniesFilter{Limit: 20},
			wantNames: []string{"Acme_100%", "Northwind Logistics"},
		},
		{
			name:      "name ignoring case",
			filter:    contacts.CompaniesFilter{Query: "NORTHWIND", Limit: 20},
			wantNames: []string{"Northwind Logistics"},
		},
		{
			name:      "literal percent",
			filter:    contacts.CompaniesFilter{Query: "%", Limit: 20},
			wantNames: []string{"Acme_100%"},
		},
		{
			name:      "literal underscore",
			filter:    contacts.CompaniesFilter{Query: "_", Limit: 20},
			wantNames: []string{"Acme_100%"},
		},
		{
			name:   "no match",
			filter: contacts.CompaniesFilter{Query: "missing", Limit: 20},
		},
		{
			name:      "paging",
			filter:    contacts.CompaniesFilter{Skip: 1, Limit: 1},
			wantNames: []string{"Northwind Logistics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := queries.ListCompanyRows(
				context.Background(),
				tt.filter,
			)
			if err != nil {
				t.Fatalf("ListCompanyRows() error = %v", err)
			}

			gotNames := make([]string, len(rows))
			for i, row := range rows {
				gotNames[i] = row.Name
			}

			if !slices.Equal(gotNames, tt.wantNames) {
				t.Fatalf(
					"ListCompanyRows() names = %v, want %v",
					gotNames,
					tt.wantNames,
				)
			}
		})
	}

	company, err := queries.GetCompany(
		context.Background(),
		common.ID(1),
	)
	if err != nil {
		t.Fatalf("GetCompany() error = %v", err)
	}

	wantCompany := contacts.CompanyView{
		ID:      common.ID(1).String(),
		Name:    "Northwind Logistics",
		Country: "Cyprus",
	}
	if company != wantCompany {
		t.Fatalf(
			"GetCompany() = %+v, want %+v",
			company,
			wantCompany,
		)
	}

	for _, id := range []common.ID{3, 999} {
		_, err := queries.GetCompany(context.Background(), id)
		if !errors.Is(err, contacts.ErrCompanyNotFound) {
			t.Fatalf(
				"GetCompany(%d) error = %v, want ErrCompanyNotFound",
				id,
				err,
			)
		}
	}
}

func TestCompanyServiceWithSQLite(t *testing.T) {
	db := openTestDatabase(t)
	repository := contacts.NewSQLiteCompanyRepository(db)
	commands := contacts.NewCompanyService(repository)
	ctx := context.Background()

	northwindID, err := commands.CreateCompany(ctx, contacts.CompanyInput{
		Name:    "  Northwind Logistics  ",
		Country: "  Cyprus  ",
	})
	if err != nil {
		t.Fatalf("CreateCompany() error = %v", err)
	}

	assertCompanyInput(
		t,
		db,
		northwindID,
		contacts.CompanyInput{
			Name:    "Northwind Logistics",
			Country: "Cyprus",
		},
	)

	_, err = commands.CreateCompany(ctx, contacts.CompanyInput{
		Name: "northwind logistics",
	})
	if !errors.Is(err, contacts.ErrCompanyNameExists) {
		t.Fatalf(
			"duplicate CreateCompany() error = %v, want ErrCompanyNameExists",
			err,
		)
	}

	acmeID, err := commands.CreateCompany(ctx, contacts.CompanyInput{
		Name:    "Acme",
		Country: "Denmark",
	})
	if err != nil {
		t.Fatalf("CreateCompany(Acme) error = %v", err)
	}

	err = commands.UpdateCompany(ctx, acmeID, contacts.CompanyInput{
		Name: "NORTHWIND LOGISTICS",
	})
	if !errors.Is(err, contacts.ErrCompanyNameExists) {
		t.Fatalf(
			"duplicate UpdateCompany() error = %v, want ErrCompanyNameExists",
			err,
		)
	}

	err = commands.UpdateCompany(ctx, northwindID, contacts.CompanyInput{
		Name:    "  Northwind Group  ",
		Country: "  France  ",
	})
	if err != nil {
		t.Fatalf("UpdateCompany() error = %v", err)
	}

	assertCompanyInput(
		t,
		db,
		northwindID,
		contacts.CompanyInput{
			Name:    "Northwind Group",
			Country: "France",
		},
	)

	if _, err := db.Exec(
		`
			INSERT INTO person (id, name, position, company)
			VALUES (101, 'Anna Petrova', 'Engineer', ?)
		`,
		northwindID.Int64(),
	); err != nil {
		t.Fatalf("insert linked person: %v", err)
	}

	if err := commands.DeleteCompany(ctx, northwindID); err != nil {
		t.Fatalf("DeleteCompany() error = %v", err)
	}

	queries := contacts.NewSQLiteCompanyQueries(db)
	if _, err := queries.GetCompany(ctx, northwindID); !errors.Is(
		err,
		contacts.ErrCompanyNotFound,
	) {
		t.Fatalf(
			"GetCompany() after delete error = %v, want ErrCompanyNotFound",
			err,
		)
	}

	personRows, err := contacts.NewSQLitePersonQueries(db).ListPersonRows(
		ctx,
		contacts.PersonsFilter{Limit: 20},
	)
	if err != nil {
		t.Fatalf("ListPersonRows() error = %v", err)
	}
	if len(personRows) != 1 || personRows[0].Company != "Northwind Group" {
		t.Fatalf(
			"person rows after company delete = %+v, want linked company",
			personRows,
		)
	}

	err = commands.UpdateCompany(ctx, northwindID, contacts.CompanyInput{
		Name: "Updated deleted company",
	})
	if !errors.Is(err, contacts.ErrCompanyNotFound) {
		t.Fatalf(
			"UpdateCompany() after delete error = %v, want ErrCompanyNotFound",
			err,
		)
	}

	err = commands.DeleteCompany(ctx, northwindID)
	if !errors.Is(err, contacts.ErrCompanyNotFound) {
		t.Fatalf(
			"second DeleteCompany() error = %v, want ErrCompanyNotFound",
			err,
		)
	}

	_, err = commands.CreateCompany(ctx, contacts.CompanyInput{
		Name: "northwind group",
	})
	if err != nil {
		t.Fatalf("reuse deleted company name: %v", err)
	}
}

func assertCompanyInput(
	t *testing.T,
	db *sql.DB,
	id common.ID,
	want contacts.CompanyInput,
) {
	t.Helper()

	var got contacts.CompanyInput
	err := db.QueryRow(
		"SELECT name, country FROM company WHERE id = ?",
		id.Int64(),
	).Scan(&got.Name, &got.Country)
	if err != nil {
		t.Fatalf("query company: %v", err)
	}

	if got != want {
		t.Fatalf("company = %+v, want %+v", got, want)
	}
}
