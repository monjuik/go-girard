package contacts

import (
	"errors"
	"strings"
	"testing"

	"github.com/monjuik/go-girard/common"
)

func TestNewPerson(t *testing.T) {
	person, err := NewPerson(
		common.ID(2),
		"  John Doe  ",
		"  Head of Operations  ",
		common.ID(1),
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
	if person.CompanyID() != common.ID(1) {
		t.Fatalf("person.CompanyID() = %d, want 1", person.CompanyID())
	}

	person, err = NewPerson(
		common.ID(3),
		"Jane Doe",
		"",
		common.ID(0),
	)
	if err != nil {
		t.Fatalf("NewPerson() without company error = %v", err)
	}
	if !person.CompanyID().IsZero() {
		t.Fatalf("person.CompanyID() = %d, want 0", person.CompanyID())
	}

	for _, id := range []common.ID{0, -1} {
		_, err = NewPerson(
			id,
			"John Doe",
			"Director",
			common.ID(0),
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
		common.ID(0),
	)
	if !errors.Is(err, ErrPersonNameRequired) {
		t.Fatalf(
			"NewPerson() error = %v, want ErrPersonNameRequired",
			err,
		)
	}

	_, err = NewPerson(
		common.ID(4),
		"John Doe",
		"Director",
		common.ID(-1),
	)
	if !errors.Is(err, ErrPersonCompanyIDInvalid) {
		t.Fatalf(
			"NewPerson() error = %v, want ErrPersonCompanyIDInvalid",
			err,
		)
	}
}

func TestPersonUpdate(t *testing.T) {
	person, err := NewPerson(
		common.ID(2),
		"John Doe",
		"Engineer",
		common.ID(1),
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
	if person.CompanyID() != common.ID(1) {
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
		person, err := NewPerson(
			common.ID(2),
			"Original Name",
			"Original Position",
			common.ID(1),
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
		if person.CompanyID() != common.ID(1) {
			t.Fatal("Update() changed company")
		}
	})
}
