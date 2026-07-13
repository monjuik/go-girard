package contacts

import (
	"testing"

	"github.com/monjuik/go-girard/common"
)

func TestNewPerson(t *testing.T) {
	company, err := NewCompany(common.ID(1), "Acme ltd.")
	if err != nil {
		t.Fatalf("NewCompany() error = %v", err)
	}

	tests := []struct {
		name       string
		id         common.ID
		personName string
		company    *Company
		wantErr    bool
	}{
		{
			name:       "valid person",
			id:         common.ID(2),
			personName: "John Doe",
			company:    &company,
			wantErr:    false,
		},
		{
			name:       "valid person without company",
			id:         common.ID(3),
			personName: "Jane Doe",
			company:    nil,
			wantErr:    false,
		},
		{
			name:       "zero id",
			id:         0,
			personName: "John Doe",
			company:    &company,
			wantErr:    true,
		},
		{
			name:       "blank name",
			id:         common.ID(4),
			personName: "   ",
			company:    &company,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			person, err := NewPerson(
				tt.id,
				tt.personName,
				"Head of Operations",
				tt.company,
				"petros.petrou@example.com",
				"+357 22 000 101",
				"Note",
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("NewPerson() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewPerson() error = %v", err)
			}

			if person.ID() != tt.id {
				t.Fatalf("person.ID() = %v, want %v", person.ID(), tt.id)
			}

			if person.Name() != tt.personName {
				t.Fatalf("person.Name() = %q, want %q", person.Name(), tt.personName)
			}

			if person.Company() != tt.company {
				t.Fatalf("person.Company() = %v, want %v", person.Company(), tt.company)
			}
		})
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
