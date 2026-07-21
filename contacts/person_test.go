package contacts

import (
	"errors"
	"strings"
	"testing"

	"github.com/monjuik/go-girard/common"
)

func TestNewPerson(t *testing.T) {
	company, err := NewCompany(common.ID(1), "Acme ltd.")
	if err != nil {
		t.Fatalf("NewCompany() error = %v", err)
	}

	person, err := NewPerson(
		common.ID(2),
		"  John Doe  ",
		"  Head of Operations  ",
		&company,
	)
	if err != nil {
		t.Fatalf("NewPerson() error = %v", err)
	}

	if person.ID() != common.ID(2) {
		t.Fatalf("person.ID() = %v, want %v", person.ID(), common.ID(2))
	}
	if person.Name() != "John Doe" {
		t.Fatalf("person.Name() = %q, want %q", person.Name(), "John Doe")
	}
	if person.Position() != "Head of Operations" {
		t.Fatalf(
			"person.Position() = %q, want %q",
			person.Position(),
			"Head of Operations",
		)
	}
	if person.Company() != &company {
		t.Fatal("person.Company() does not contain the provided company")
	}

	person, err = NewPerson(
		common.ID(3),
		"Jane Doe",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("NewPerson() without company error = %v", err)
	}
	if person.Company() != nil {
		t.Fatal("person.Company() != nil")
	}

	for _, id := range []common.ID{0, -1} {
		_, err = NewPerson(
			id,
			"John Doe",
			"Director",
			nil,
		)
		if !errors.Is(err, ErrPersonIDInvalid) {
			t.Fatalf(
				"NewPerson(%d) error = %v, want ErrPersonIDInvalid",
				id,
				err,
			)
		}
	}

	_, err = NewPerson(
		common.ID(4),
		" \t ",
		"Director",
		nil,
	)
	if !errors.Is(err, ErrPersonNameRequired) {
		t.Fatalf(
			"NewPerson() error = %v, want ErrPersonNameRequired",
			err,
		)
	}
}

func TestNewCompany(t *testing.T) {
	tests := []struct {
		name        string
		id          common.ID
		companyName string
		wantErr     bool
	}{
		{
			name:        "valid company",
			id:          common.ID(1),
			companyName: "Acme ltd.",
			wantErr:     false,
		},
		{
			name:        "zero id",
			id:          0,
			companyName: "Acme ltd.",
			wantErr:     true,
		},
		{
			name:        "blank name",
			id:          common.ID(2),
			companyName: "   ",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			company, err := NewCompany(tt.id, tt.companyName)

			if tt.wantErr {
				if err == nil {
					t.Fatal("NewCompany() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewCompany() error = %v", err)
			}

			if company.ID() != tt.id {
				t.Fatalf("company.ID() = %v, want %v", company.ID(), tt.id)
			}

			if company.Name() != tt.companyName {
				t.Fatalf("company.Name() = %q, want %q", company.Name(), tt.companyName)
			}
		})
	}
}

func TestPersonUpdate(t *testing.T) {
	company, err := NewCompany(common.ID(1), "Acme ltd.")
	if err != nil {
		t.Fatalf("NewCompany() error = %v", err)
	}

	person, err := NewPerson(
		common.ID(2),
		"John Doe",
		"Engineer",
		&company,
	)
	if err != nil {
		t.Fatalf("NewPerson() error = %v", err)
	}

	err = person.Update("  Jane Doe  ", "  Director  ")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if person.Name() != "Jane Doe" {
		t.Fatalf("person.Name() = %q, want %q", person.Name(), "Jane Doe")
	}
	if person.Position() != "Director" {
		t.Fatalf("person.Position() = %q, want %q", person.Position(), "Director")
	}
	if person.Company() != &company {
		t.Fatal("Update() changed company")
	}

	err = person.Update(" \t ", "Invalid")
	if !errors.Is(err, ErrPersonNameRequired) {
		t.Fatalf("Update() error = %v, want ErrPersonNameRequired", err)
	}

	if person.Name() != "Jane Doe" || person.Position() != "Director" {
		t.Fatal("failed Update() changed person")
	}
}

func FuzzPersonUpdate(f *testing.F) {
	for _, seed := range []struct {
		name     string
		position string
	}{
		{"Jane Doe", "Director"},
		{"  Jane Doe  ", "  Director  "},
		{"", "Director"},
		{" \t\n", "Director"},
		{"  Мария 李  ", "  Инженер  "},
		{"\u00a0", "Director"},
		{"<script>alert(1)</script>", "%_&=+"},
	} {
		f.Add(seed.name, seed.position)
	}

	f.Fuzz(func(t *testing.T, name, position string) {
		company, err := NewCompany(common.ID(1), "Acme ltd.")
		if err != nil {
			t.Fatalf("NewCompany() error = %v", err)
		}

		person, err := NewPerson(
			common.ID(2),
			"Original Name",
			"Original Position",
			&company,
		)
		if err != nil {
			t.Fatalf("NewPerson() error = %v", err)
		}

		err = person.Update(name, position)

		wantName := strings.TrimSpace(name)
		if wantName == "" {
			if !errors.Is(err, ErrPersonNameRequired) {
				t.Fatalf(
					"Update(%q, %q) error = %v, want ErrPersonNameRequired",
					name,
					position,
					err,
				)
			}

			if person.Name() != "Original Name" ||
				person.Position() != "Original Position" {
				t.Fatal("failed Update() changed person")
			}

			return
		}

		if err != nil {
			t.Fatalf("Update(%q, %q) error = %v", name, position, err)
		}

		if person.Name() != wantName {
			t.Fatalf("person.Name() = %q, want %q", person.Name(), wantName)
		}

		wantPosition := strings.TrimSpace(position)
		if person.Position() != wantPosition {
			t.Fatalf(
				"person.Position() = %q, want %q",
				person.Position(),
				wantPosition,
			)
		}

		if person.ID() != common.ID(2) {
			t.Fatalf("Update() changed person ID to %d", person.ID())
		}
		if person.Company() != &company {
			t.Fatal("Update() changed company")
		}
	})
}
